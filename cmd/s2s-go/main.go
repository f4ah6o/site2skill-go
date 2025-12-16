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

	"github.com/f4ah6o/site2skill-go/internal/converter"
	"github.com/f4ah6o/site2skill-go/internal/fetcher"
	"github.com/f4ah6o/site2skill-go/internal/normalizer"
	"github.com/f4ah6o/site2skill-go/internal/packager"
	"github.com/f4ah6o/site2skill-go/internal/skillgen"
	"github.com/f4ah6o/site2skill-go/internal/validator"
)

const (
	// FormatClaude specifies output format for Claude AI skill packages.
	FormatClaude = "claude"
	// FormatCodex specifies output format for OpenAI Codex skill packages.
	FormatCodex = "codex"
)

func main() {
	// Command line flags
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

	flag.StringVar(&url, "url", "", "URL of the documentation site (required)")
	flag.StringVar(&skillName, "name", "", "Name of the skill (required)")
	flag.StringVar(&output, "output", ".claude/skills", "Base output directory for skill structure")
	flag.StringVar(&skillOutput, "skill-output", ".", "Output directory for .skill file")
	flag.StringVar(&tempDir, "temp-dir", "build", "Temporary directory for processing")
	flag.BoolVar(&skipFetch, "skip-fetch", false, "Skip the download step (use existing files in temp dir)")
	flag.BoolVar(&clean, "clean", false, "Clean up temporary directory after completion")
	flag.StringVar(&format, "format", "claude", "Output format: claude or codex")

	flag.Parse()

	// Handle positional arguments if provided
	if flag.NArg() >= 2 {
		url = flag.Arg(0)
		skillName = flag.Arg(1)
	}

	if url == "" || skillName == "" {
		fmt.Fprintf(os.Stderr, "Usage: s2s-go <URL> <SKILL_NAME> [options]\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Validate format
	if format != FormatClaude && format != FormatCodex {
		log.Fatalf("Invalid format: %s. Must be 'claude' or 'codex'", format)
	}

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
