// Package main is the entry point for the site2skillgo tool.
// site2skillgo converts website documentation into Claude/Codex AI skill packages
// through a multi-step pipeline: fetch, convert, normalize, generate, validate, and package.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/f4ah6o/site2skill-go/internal/converter"
	"github.com/f4ah6o/site2skill-go/internal/fetcher"
	"github.com/f4ah6o/site2skill-go/internal/normalizer"
	"github.com/f4ah6o/site2skill-go/internal/packager"
	"github.com/f4ah6o/site2skill-go/internal/search"
	"github.com/f4ah6o/site2skill-go/internal/skillgen"
	"github.com/f4ah6o/site2skill-go/internal/validator"
)

const (
	// FormatClaude specifies output format for Claude AI skill packages.
	FormatClaude = "claude"
	// FormatCodex specifies output format for OpenAI Codex skill packages.
	FormatCodex = "codex"
	// FormatBoth specifies output format for both Claude and Codex skill packages.
	FormatBoth = "both"
)

// CodexConfig represents the structure of config.toml
type CodexConfig struct {
	Features struct {
		Skills bool `toml:"skills"`
	} `toml:"features"`
}

// getCodexHome returns the Codex home directory.
// It checks the CODEX_HOME environment variable first, then falls back to ~/.codex
func getCodexHome() (string, error) {
	if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
		return codexHome, nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	return filepath.Join(usr.HomeDir, ".codex"), nil
}

// checkCodexSkillsConfig checks if Codex skills feature is enabled in config.toml
// Returns: (enabled bool, configExists bool, err error)
func checkCodexSkillsConfig() (bool, bool, error) {
	codexHome, err := getCodexHome()
	if err != nil {
		return false, false, err
	}

	configPath := filepath.Join(codexHome, "config.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, false, nil
	}

	var config CodexConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return false, true, fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	return config.Features.Skills, true, nil
}

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
			log.Println("Note: Using legacy command format. Consider using 'site2skillgo generate' subcommand.")
			runGenerate(os.Args[1:])
		} else {
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", subcommand)
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `site2skillgo - Convert website documentation into AI skill packages

site2skillgo is a tool that scrapes documentation websites, converts them to Markdown,
and packages them as Agent Skills (ZIP format) for use with Claude or Codex.

Usage:
  site2skillgo generate <URL> <SKILL_NAME> [options]
  site2skillgo search <QUERY> [options]
  site2skillgo help

Commands:
  generate    Generate a skill package from a documentation website
  search      Search through skill documentation files
  help        Show this help message

Examples:
  site2skillgo generate https://docs.example.com myskill
  site2skillgo generate https://stripe.com/docs/api stripe --format codex
  site2skillgo search "authentication" --skill-dir .claude/skills/myskill

For more information on a command, use:
  site2skillgo <command> -h

Supported Formats:
  claude      Claude Agent Skills (default, optimized for Claude SDK)
  codex       OpenAI Codex Skills (optimized for OpenAI Codex)
`)
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)

	var (
		url              string
		skillName        string
		global           bool
		tempDir          string
		skipFetch        bool
		clean            bool
		format           string
		localePriority   string
		noLocalePriority bool
		localeParam      string
	)

	fs.StringVar(&url, "url", "", "URL of the documentation site (required)")
	fs.StringVar(&skillName, "name", "", "Name of the skill (required)")
	fs.BoolVar(&global, "global", false, "Install to global skills directory (~/.claude/skills or ~/.codex/skills)")
	fs.StringVar(&tempDir, "temp-dir", "build", "Temporary directory for processing")
	fs.BoolVar(&skipFetch, "skip-fetch", false, "Skip the download step (use existing files in temp dir)")
	fs.BoolVar(&clean, "clean", false, "Clean up temporary directory after completion")
	fs.StringVar(&format, "format", "claude", "Output format: claude, codex, or both")
	fs.StringVar(&localePriority, "locale-priority", "en,ja", "Locale priority order (comma-separated, e.g., 'en,ja,zh')")
	fs.BoolVar(&noLocalePriority, "no-locale-priority", false, "Disable locale priority mode")
	fs.StringVar(&localeParam, "locale-param", "", "Query parameter name for locale (e.g., 'hl' for ?hl=ja)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: site2skillgo generate <URL> <SKILL_NAME> [options]

Generate a skill package from a documentation website.

Arguments:
  URL           URL of the documentation site to scrape
  SKILL_NAME    Name for the generated skill

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Pipeline Steps:
  1. Fetch      - Download documentation site recursively
  2. Convert    - Convert HTML pages to Markdown
  3. Normalize  - Clean up links and formatting
  4. Generate   - Create skill structure
  5. Validate   - Check skill structure and size limits
  6. Package    - Create .skill file (ZIP archive)

Examples:
  site2skillgo generate https://docs.example.com example
  site2skillgo generate https://docs.python.org/3/ python3 --format claude
  site2skillgo generate https://stripe.com/docs/api stripe --format codex --global
  site2skillgo generate https://docs.example.com example --skip-fetch --clean
`)
	}

	fs.Parse(args)

	// Handle positional arguments if provided
	if fs.NArg() >= 2 {
		url = fs.Arg(0)
		skillName = fs.Arg(1)
	}

	if url == "" || skillName == "" {
		fmt.Fprintf(os.Stderr, "Usage: site2skillgo generate <URL> <SKILL_NAME> [options]\n\n")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Validate format
	if format != FormatClaude && format != FormatCodex && format != FormatBoth {
		log.Fatalf("Invalid format: %s. Must be 'claude', 'codex', or 'both'", format)
	}

	executeGenerate(url, skillName, global, tempDir, skipFetch, clean, format, localePriority, noLocalePriority, localeParam)
}

// determineOutputPaths determines the output directories based on format and scope (global/local)
func determineOutputPaths(format string, global bool) (skillStructureDir, skillFileDir string) {
	if global {
		if format == FormatClaude {
			usr, err := user.Current()
			if err != nil {
				log.Fatalf("Failed to get current user: %v", err)
			}
			skillStructureDir = filepath.Join(usr.HomeDir, ".claude", "skills")
			skillFileDir = skillStructureDir
		} else if format == FormatCodex {
			codexHome, err := getCodexHome()
			if err != nil {
				log.Fatalf("Failed to get Codex home directory: %v", err)
			}
			skillStructureDir = filepath.Join(codexHome, "skills")
			skillFileDir = skillStructureDir
		}
	} else {
		// local (default)
		if format == FormatClaude {
			skillStructureDir = filepath.Join(".claude", "skills")
			skillFileDir = skillStructureDir
		} else if format == FormatCodex {
			skillStructureDir = filepath.Join(".codex", "skills")
			skillFileDir = skillStructureDir
		}
	}

	return skillStructureDir, skillFileDir
}

func executeGenerate(url, skillName string, global bool, tempDir string, skipFetch, clean bool, format, localePriority string, noLocalePriority bool, localeParam string) {
	// Check Codex skills configuration if generating codex format
	if format == FormatCodex || format == FormatBoth {
		enabled, configExists, err := checkCodexSkillsConfig()
		if err != nil {
			log.Printf("Warning: %v", err)
		} else if !configExists {
			codexHome, _ := getCodexHome()
			configPath := filepath.Join(codexHome, "config.toml")
			log.Printf("Info: %s not found. To enable Codex skills, create the file with:", configPath)
			log.Printf("  [features]")
			log.Printf("  skills = true")
		} else if !enabled {
			codexHome, _ := getCodexHome()
			configPath := filepath.Join(codexHome, "config.toml")
			log.Printf("Info: Codex skills feature is not enabled. To enable it, add the following to %s:", configPath)
			log.Printf("  [features]")
			log.Printf("  skills = true")
		}
	}

	// Determine output directories based on format and global flag
	var output, skillOutput string

	if format == FormatBoth {
		// For both format, we'll handle each format separately in the generation step
		// Set a placeholder for now
		output = ""
		skillOutput = ""
	} else {
		output, skillOutput = determineOutputPaths(format, global)
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

		// Configure locale priority if enabled
		if !noLocalePriority {
			locales := parseLocales(localePriority)
			cfg := &fetcher.LocaleConfig{
				Priority:  locales,
				ParamName: localeParam,
			}
			f.SetLocaleConfig(cfg)
			log.Printf("Locale priority mode enabled: %v", locales)
			if localeParam != "" {
				log.Printf("Using query parameter: ?%s=<locale>", localeParam)
			}
		}

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
	type skillInfo struct {
		dir        string
		outputPath string
	}
	var skills []skillInfo

	if format == FormatBoth {
		// Generate both Claude and Codex formats
		for _, fmt := range []string{FormatClaude, FormatCodex} {
			log.Printf("=== Step 4: Generating Skill Structure (%s format) ===", fmt)
			fmtOutput, fmtSkillOutput := determineOutputPaths(fmt, global)
			gen := skillgen.New(fmt)
			if err := gen.Generate(skillName, tempMdDir, fmtOutput); err != nil {
				log.Fatalf("Failed to generate skill structure: %v", err)
			}
			skills = append(skills, skillInfo{
				dir:        filepath.Join(fmtOutput, skillName),
				outputPath: fmtSkillOutput,
			})
		}
	} else {
		log.Printf("=== Step 4: Generating Skill Structure (%s format) ===", format)
		gen := skillgen.New(format)
		if err := gen.Generate(skillName, tempMdDir, output); err != nil {
			log.Fatalf("Failed to generate skill structure: %v", err)
		}
		skills = append(skills, skillInfo{
			dir:        filepath.Join(output, skillName),
			outputPath: skillOutput,
		})
	}

	// Step 5: Validate Skill
	log.Printf("=== Step 5: Validating Skill ===")
	val := validator.New()
	for _, skill := range skills {
		if !val.Validate(skill.dir) {
			log.Printf("Warning: Validation failed for %s. Please check errors.", skill.dir)
		}
	}

	// Step 6: Package Skill
	log.Printf("=== Step 6: Packaging Skill ===")
	pkg := packager.New()
	var skillFiles []string
	for _, skill := range skills {
		skillFile, err := pkg.Package(skill.dir, skill.outputPath)
		if err != nil {
			log.Fatalf("Failed to package skill: %v", err)
		}
		skillFiles = append(skillFiles, skillFile)
	}

	log.Printf("=== Done! ===")
	for i, file := range skillFiles {
		log.Printf("Skill package %d: %s", i+1, file)
	}

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

// parseLocales parses a comma-separated locale string into a slice
func parseLocales(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, part := range splitByComma(s) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// splitByComma splits a string by commas
func splitByComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// trimSpace trims leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
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
		fmt.Fprintf(os.Stderr, `Usage: site2skillgo search <QUERY> [options]

Search through skill documentation files for keywords.

Arguments:
  QUERY       Search query (space-separated keywords with OR logic)

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  site2skillgo search "authentication"
  site2skillgo search "api endpoint" --max-results 5
  site2skillgo search "database" --json --skill-dir ./my-skill
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
