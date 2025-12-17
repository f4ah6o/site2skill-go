// Package normalizer provides Markdown file normalization functionality.
// It processes YAML frontmatter, resolves relative links to absolute URLs,
// and reconstructs normalized Markdown files with updated metadata.
package normalizer

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Normalizer processes and normalizes Markdown files generated from HTML documentation.
// It extracts YAML frontmatter, resolves relative links to absolute URLs, and reconstructs
// the Markdown with updated metadata. This ensures all documentation files have consistent
// formatting and functional links.
type Normalizer struct{}

// New creates a new Normalizer instance.
//
// Returns a Normalizer ready to process Markdown files.
func New() *Normalizer {
	return &Normalizer{}
}

// Frontmatter represents the YAML frontmatter metadata of a Markdown documentation file.
// This metadata is added during the HTML-to-Markdown conversion process and provides
// important context about the document's origin and freshness.
type Frontmatter struct {
	// Title is the document title extracted from the HTML <title> or <h1> tag.
	Title string `yaml:"title"`
	// SourceURL is the original URL where the document was fetched from.
	// Used for citation and to enable absolute link resolution.
	SourceURL string `yaml:"source_url"`
	// FetchedAt is the ISO 8601 timestamp when the document was fetched.
	// Format: "2006-01-02T15:04:05Z07:00"
	FetchedAt string `yaml:"fetched_at"`
}

// NormalizeFile normalizes a Markdown documentation file by processing its frontmatter
// and converting all relative links to absolute URLs.
//
// The normalization process:
//   1. Reads the Markdown file
//   2. Extracts and parses YAML frontmatter
//   3. Resolves all relative Markdown links [text](url) to absolute URLs using source_url as the base
//   4. Preserves absolute URLs and anchor links unchanged
//   5. Reconstructs the file with updated frontmatter and normalized content
//   6. Writes the result to the output path
//
// Link resolution examples (assuming source_url: https://example.com/docs/api.html):
//   [guide](../guide.html) -> [guide](https://example.com/guide.html)
//   [home](/) -> [home](https://example.com/)
//   [section](#header) -> [section](#header) (unchanged)
//   [external](https://other.com) -> [external](https://other.com) (unchanged)
//
// Parameters:
//   - inputPath: Path to the Markdown file to normalize
//   - outputPath: Path where the normalized file will be written (can be the same as inputPath)
//
// Returns an error if the file cannot be read, parsed, or written.
func (n *Normalizer) NormalizeFile(inputPath, outputPath string) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Extract frontmatter
	frontmatter, body, err := n.extractFrontmatter(string(content))
	if err != nil {
		return fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// Normalize links if we have a source URL
	if frontmatter != nil && frontmatter.SourceURL != "" {
		body = n.normalizeLinks(body, frontmatter.SourceURL)
	}

	// Reconstruct content
	var finalContent string
	if frontmatter != nil {
		yamlBytes, _ := yaml.Marshal(frontmatter)
		finalContent = "---\n" + string(yamlBytes) + "---\n\n" + body
	} else {
		finalContent = body
	}

	// Write output
	if err := os.WriteFile(outputPath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractFrontmatter parses YAML frontmatter from Markdown content.
// It extracts the metadata section between --- delimiters and returns the frontmatter struct and remaining body.
func (n *Normalizer) extractFrontmatter(content string) (*Frontmatter, string, error) {
	// Match YAML frontmatter
	re := regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)$`)
	matches := re.FindStringSubmatch(content)

	if len(matches) != 3 {
		return nil, content, nil
	}

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(matches[1]), &fm); err != nil {
		return nil, content, err
	}

	return &fm, matches[2], nil
}

// normalizeLinks resolves all relative Markdown links in content to absolute URLs.
// It uses the sourceURL as the base to resolve relative references and preserves absolute URLs unchanged.
func (n *Normalizer) normalizeLinks(content, sourceURL string) string {
	// Regex to capture markdown links: [text](url)
	re := regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		text := submatches[1]
		linkURL := submatches[2]

		// Skip if already absolute
		if strings.HasPrefix(linkURL, "http:") ||
			strings.HasPrefix(linkURL, "https:") ||
			strings.HasPrefix(linkURL, "mailto:") {
			return match
		}

		// Skip anchors only
		if strings.HasPrefix(linkURL, "#") {
			return match
		}

		// Resolve absolute URL
		base, err := url.Parse(sourceURL)
		if err != nil {
			return match
		}

		rel, err := url.Parse(linkURL)
		if err != nil {
			return match
		}

		absoluteURL := base.ResolveReference(rel)
		return fmt.Sprintf("[%s](%s)", text, absoluteURL.String())
	})
}
