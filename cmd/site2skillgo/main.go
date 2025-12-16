// Package main is the entry point for the s2s-go tool.
// s2s-go converts website documentation into Claude/Codex AI skill packages
// through a multi-step pipeline: fetch, convert, normalize, generate, validate, and package.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/f4ah6o/site2skillgo/internal/converter"
	"github.com/f4ah6o/site2skillgo/internal/fetcher"
	"github.com/f4ah6o/site2skillgo/internal/normalizer"
	"github.com/f4ah6o/site2skillgo/internal/packager"
	"github.com/f4ah6o/site2skillgo/internal/search"
	"github.com/f4ah6o/site2skillgo/internal/skillgen"
	"github.com/f4ah6o/site2skillgo/internal/validator"
)

const (
	// FormatClaude specifies output format for Claude AI skill packages.
	FormatClaude = "claude"
	// FormatCodex specifies output format for OpenAI Codex skill packages.
	FormatCodex = "codex"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "generate":
		runGenerate(os.Args[2:])
	case "search":
		runSearch(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
	default:
		// Try to parse as old-style command (backwards compatibility)
		// If first arg looks like a URL, treat as generate command
		if len(os.Args) >= 3 && (len(subcommand) > 4 && subcommand[:4] == "http") {
			log.Println("Note: Using legacy command format. Consider using 's2s-go generate' subcommand.")
			runGenerate(os.Args[1:])
		} else {
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", subcommand)
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `s2s-go - Convert website documentation into AI skill packages

Usage:
  s2s-go generate <URL> <SKILL_NAME> [options]
  s2s-go search <QUERY> [options]
  s2s-go help

Commands:
  generate    Generate a skill package from a documentation website
  search      Search through skill documentation files
  help        Show this help message

Run 's2s-go <command> -h' for more information on a specific command.
`)
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)

	var (
		url          string
		skillName    string
		output       string
		skillOutput  string
		tempDir      string
		skipFetch    bool
		clean        bool
		format       string
	)

	fs.StringVar(&url, "url", "", "URL of the documentation site (required)")
	fs.StringVar(&skillName, "name", "", "Name of the skill (required)")
	fs.StringVar(&output, "output", ".claude/skills", "Base output directory for skill structure")
	fs.StringVar(&skillOutput, "skill-output", ".", "Output directory for .skill file")
	fs.StringVar(&tempDir, "temp-dir", "build", "Temporary directory for processing")
	fs.BoolVar(&skipFetch, "skip-fetch", false, "Skip the download step (use existing files in temp dir)")
	fs.BoolVar(&clean, "clean", false, "Clean up temporary directory after completion")
	fs.StringVar(&format, "format", "claude", "Output format: claude or codex")

	fs.Parse(args)

	// Handle positional arguments if provided
	if fs.NArg() >= 2 {
		url = fs.Arg(0)
		skillName = fs.Arg(1)
	}

	if url == "" || skillName == "" {
		fmt.Fprintf(os.Stderr, "Usage: s2s-go generate <URL> <SKILL_NAME> [options]\n\n")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Validate format
	if format != FormatClaude && format != FormatCodex {
		log.Fatalf("Invalid format: %s. Must be 'claude' or 'codex'", format)
	}

	executeGenerate(url, skillName, output, skillOutput, tempDir, skipFetch, clean, format)
}

func executeGenerate(url, skillName, output, skillOutput, tempDir string, skipFetch, clean bool, format string) {
	// Setup directories
	tempDownloadDir := filepath.Join(tempDir, "download")
	tempMdDir := filepath.Join(tempDir, "markdown")

	if !skipFetch {
		if err := os.RemoveAll(tempDir); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: could not remove temp dir: %v", err)
		}
		if err := os.MkdirAll(tempDownloadDir, 0755); err != nil {
			log.Fatalf("Failed to create temp download dir: %v", err)
		}
	}

	if err := os.MkdirAll(tempMdDir, 0755); err != nil {
		log.Fatalf("Failed to create temp markdown dir: %v", err)
	}

	fetchedAt := time.Now().UTC().Format(time.RFC3339)

	// Step 1: Fetch
	if !skipFetch {
		log.Printf("=== Step 1: Fetching %s ===", url)
		f := fetcher.New(tempDownloadDir)
		if err := f.Fetch(url); err != nil {
			log.Fatalf("Failed to fetch site: %v", err)
		}
	} else {
		log.Printf("=== Step 1: Skipped Fetching (Using %s) ===", tempDownloadDir)
	}

	crawlDir := filepath.Join(tempDownloadDir, "crawl")

	// Step 2: Convert HTML to Markdown
	log.Printf("=== Step 2: Converting HTML to Markdown ===")
	htmlFiles, err := filepath.Glob(filepath.Join(crawlDir, "**/*.html"))
	if err != nil {
		log.Fatalf("Failed to find HTML files: %v", err)
	}

	// Better way to find all HTML files recursively
	htmlFiles = []string{}
	filepath.Walk(crawlDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".html" {
			htmlFiles = append(htmlFiles, path)
		}
		return nil
	})

	log.Printf("Found %d HTML files.", len(htmlFiles))

	conv := converter.New()
	for _, htmlFile := range htmlFiles {
		// Security check
		absHTMLFile, err := filepath.Abs(htmlFile)
		if err != nil {
			log.Printf("Warning: could not get absolute path for %s: %v", htmlFile, err)
			continue
		}
		absCrawlDir, err := filepath.Abs(crawlDir)
		if err != nil {
			log.Printf("Warning: could not get absolute path for crawl dir: %v", err)
			continue
		}

		relPath, err := filepath.Rel(absCrawlDir, absHTMLFile)
		if err != nil || len(relPath) > 0 && relPath[0] == '.' {
			log.Printf("Warning: skipping potential path traversal file: %s", htmlFile)
			continue
		}

		// Construct source URL
		sourceURL := reconstructURL(url, relPath)

		// Determine output filename
		baseName := filepath.Base(htmlFile)
		nameWithoutExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]
		mdFilename := sanitizeFilename(nameWithoutExt) + ".md"
		mdPath := filepath.Join(tempMdDir, mdFilename)

		if _, err := os.Stat(mdPath); err == nil {
			log.Printf("Warning: name collision for %s. Overwriting.", mdFilename)
		}

		if err := conv.ConvertFile(htmlFile, mdPath, sourceURL, fetchedAt); err != nil {
			log.Printf("Error converting %s: %v", htmlFile, err)
		}
	}

	// Step 3: Normalize Markdown
	log.Printf("=== Step 3: Normalizing Markdown ===")
	mdFiles, err := filepath.Glob(filepath.Join(tempMdDir, "*.md"))
	if err != nil {
		log.Fatalf("Failed to find markdown files: %v", err)
	}

	norm := normalizer.New()
	for _, mdFile := range mdFiles {
		if err := norm.NormalizeFile(mdFile, mdFile); err != nil {
			log.Printf("Error normalizing %s: %v", mdFile, err)
		}
	}

	// Step 4: Generate Skill Structure
	log.Printf("=== Step 4: Generating Skill Structure (%s format) ===", format)
	gen := skillgen.New(format)
	if err := gen.Generate(skillName, tempMdDir, output); err != nil {
		log.Fatalf("Failed to generate skill structure: %v", err)
	}

	skillDir := filepath.Join(output, skillName)

	// Step 5: Validate Skill
	log.Printf("=== Step 5: Validating Skill ===")
	val := validator.New()
	if !val.Validate(skillDir) {
		log.Printf("Warning: Validation failed. Please check errors.")
	}

	// Step 6: Package Skill
	log.Printf("=== Step 6: Packaging Skill ===")
	pkg := packager.New()
	skillFile, err := pkg.Package(skillDir, skillOutput)
	if err != nil {
		log.Fatalf("Failed to package skill: %v", err)
	}

	log.Printf("=== Done! ===")
	log.Printf("Skill directory: %s", skillDir)
	log.Printf("Skill package: %s", skillFile)

	// Cleanup
	if clean {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Printf("Warning: could not remove temp dir: %v", err)
		}
		log.Printf("Temporary files removed from %s", tempDir)
	} else {
		log.Printf("Temporary files kept in %s", tempDir)
	}
}

// reconstructURL reconstructs the original URL from the crawl path
func reconstructURL(baseURL, relPath string) string {
	// Remove .html extension if present
	if len(relPath) > 5 && relPath[len(relPath)-5:] == ".html" {
		relPath = relPath[:len(relPath)-5]
	}

	// Parse base URL to get scheme
	scheme := "https"
	if len(baseURL) > 7 && baseURL[:7] == "http://" {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s", scheme, relPath)
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	// Replace non-alphanumeric characters (except ._-) with _
	result := ""
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '_' || ch == '-' {
			result += string(ch)
		} else {
			result += "_"
		}
	}
	return result
}

func runSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)

	var (
		skillDir   string
		maxResults int
		jsonOutput bool
	)

	fs.StringVar(&skillDir, "skill-dir", ".", "Path to the skill directory")
	fs.IntVar(&maxResults, "max-results", 10, "Maximum number of results to display")
	fs.BoolVar(&jsonOutput, "json", false, "Output results as JSON")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: s2s-go search <QUERY> [options]

Search through skill documentation files for keywords.

Arguments:
  QUERY       Search query (space-separated keywords with OR logic)

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  s2s-go search "authentication"
  s2s-go search "api endpoint" --max-results 5
  s2s-go search "database" --json --skill-dir ./my-skill
`)
	}

	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: search query is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	query := fs.Arg(0)

	opts := search.SearchOptions{
		SkillDir:   skillDir,
		Query:      query,
		MaxResults: maxResults,
		JSONOutput: jsonOutput,
	}

	results, err := search.SearchDocs(opts)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	if jsonOutput {
		if err := search.FormatJSON(results); err != nil {
			log.Fatalf("Failed to format JSON output: %v", err)
		}
	} else {
		search.FormatResults(results, query)
	}
}
