package search

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
)

const (
	contextLines = 2
)

var (
	// ANSI colors for terminal output
	colorHeader  = color.New(color.FgHiMagenta, color.Bold)
	colorBold    = color.New(color.Bold)
	colorCyan    = color.New(color.FgCyan)
	colorWarning = color.New(color.FgYellow)
)

// extractFrontmatter parses YAML frontmatter from Markdown content
func extractFrontmatter(content string) (Frontmatter, string) {
	fm := Frontmatter{
		SourceURL: "Unknown",
		FetchedAt: "Unknown",
	}

	// Match frontmatter: ---\n...\n---\n
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)`)
	matches := re.FindStringSubmatch(content)

	if len(matches) < 3 {
		return fm, content
	}

	frontmatterStr := matches[1]
	body := matches[2]

	// Parse simple YAML key-value pairs
	lines := strings.Split(frontmatterStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

				switch strings.ToLower(key) {
				case "title":
					fm.Title = value
				case "source_url":
					fm.SourceURL = value
				case "fetched_at":
					fm.FetchedAt = value
				}
			}
		}
	}

	return fm, body
}

// getContext finds matches and extracts surrounding context lines
func getContext(text, query string) []string {
	lines := strings.Split(text, "\n")
	keywords := strings.Fields(strings.ToLower(query))
	var contexts []string

	// Find all matching line indices
	var matchIndices []int
	for i, line := range lines {
		lineLower := strings.ToLower(line)
		for _, kw := range keywords {
			if strings.Contains(lineLower, kw) {
				matchIndices = append(matchIndices, i)
				break
			}
		}
	}

	if len(matchIndices) == 0 {
		return contexts
	}

	// Group nearby matches
	var groups [][]int
	currentGroup := []int{matchIndices[0]}

	for i := 1; i < len(matchIndices); i++ {
		if matchIndices[i]-matchIndices[i-1] <= (contextLines*2 + 1) {
			currentGroup = append(currentGroup, matchIndices[i])
		} else {
			groups = append(groups, currentGroup)
			currentGroup = []int{matchIndices[i]}
		}
	}
	groups = append(groups, currentGroup)

	// Extract context for each group
	for _, group := range groups {
		startIdx := max(0, group[0]-contextLines)
		endIdx := min(len(lines), group[len(group)-1]+contextLines+1)

		var snippetLines []string
		for i := startIdx; i < endIdx; i++ {
			prefix := "  "
			// Check if this line is a match line
			for _, matchIdx := range group {
				if i == matchIdx {
					prefix = "> "
					break
				}
			}
			snippetLines = append(snippetLines, prefix+lines[i])
		}

		contexts = append(contexts, strings.Join(snippetLines, "\n"))
	}

	return contexts
}

// SearchDocs searches documentation files in the skill directory
func SearchDocs(opts SearchOptions) ([]SearchResult, error) {
	// Convert to absolute path
	absSkillDir, err := filepath.Abs(opts.SkillDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	docsDir := filepath.Join(absSkillDir, "docs")
	if _, statErr := os.Stat(docsDir); os.IsNotExist(statErr) {
		return nil, fmt.Errorf("docs directory not found: %s (skill dir: %s)", docsDir, absSkillDir)
	}

	keywords := strings.Fields(strings.ToLower(opts.Query))
	var results []SearchResult

	// Walk through all .md files in docs directory
	walkErr := filepath.Walk(docsDir, func(path string, info os.FileInfo, walkFuncErr error) error {
		if walkFuncErr != nil {
			return walkFuncErr
		}
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
			return nil // Continue with other files
		}

		// Extract frontmatter and body
		frontmatter, body := extractFrontmatter(string(content))
		bodyLower := strings.ToLower(body)

		// Count matches
		matchesCount := 0
		for _, kw := range keywords {
			matchesCount += strings.Count(bodyLower, kw)
		}

		if matchesCount > 0 {
			contexts := getContext(body, opts.Query)

			relPath, _ := filepath.Rel(absSkillDir, path)
			results = append(results, SearchResult{
				File:      relPath,
				Matches:   matchesCount,
				Contexts:  contexts,
				SourceURL: frontmatter.SourceURL,
				FetchedAt: frontmatter.FetchedAt,
			})
		}

		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	// Sort by matches count (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Matches > results[j].Matches
	})

	// Limit results
	if opts.MaxResults > 0 && len(results) > opts.MaxResults {
		results = results[:opts.MaxResults]
	}

	return results, nil
}

// FormatResults prints results in a human-readable format
func FormatResults(results []SearchResult, query string) {
	if len(results) == 0 {
		fmt.Printf("No matches found for '%s'.\n", query)
		return
	}

	colorHeader.Printf("\nSearch Results for '%s'\n", query)
	fmt.Printf("Found matches in %d files.\n\n", len(results))

	for i, res := range results {
		colorBold.Printf("%d. %s\n", i+1, res.File)
		fmt.Printf("   Matches: %d | Source: %s\n", res.Matches, res.SourceURL)
		fmt.Printf("   Fetched: %s\n", res.FetchedAt)
		colorCyan.Println(strings.Repeat("-", 40))

		// Show up to 3 contexts
		maxContexts := min(3, len(res.Contexts))
		for j := 0; j < maxContexts; j++ {
			fmt.Println(res.Contexts[j])
			if j < maxContexts-1 || len(res.Contexts) > 3 {
				fmt.Println("   ...")
			}
		}
		fmt.Println()
	}
}

// FormatJSON prints results as JSON
func FormatJSON(results []SearchResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
