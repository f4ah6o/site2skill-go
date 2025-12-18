// Package converter provides HTML to Markdown conversion functionality.
// It extracts main content from HTML pages, cleans unwanted elements,
// and converts to Markdown with YAML frontmatter containing metadata.
package converter

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// Converter converts HTML content to Markdown format with YAML frontmatter metadata.
// It extracts main content from HTML documents, cleans unwanted elements, and generates
// Markdown suitable for documentation skill packages. The converter handles character
// encoding detection and preserves document structure.
type Converter struct {
	// mdConverter is the underlying HTML to Markdown conversion engine
	mdConverter *md.Converter
}

// New creates a new Converter instance with default configuration.
// The converter is configured to preserve links and use standard Markdown formatting.
//
// Returns a Converter ready to process HTML files.
func New() *Converter {
	converter := md.NewConverter("", true, nil)
	return &Converter{
		mdConverter: converter,
	}
}

// ConvertFile converts an HTML file to Markdown with YAML frontmatter metadata.
// It performs the following steps:
//  1. Reads and decodes the HTML file with proper charset handling
//  2. Parses the HTML and extracts the title (from <title> or <h1>)
//  3. Identifies and extracts main content (from <main>, <article>, <div.content>, or <body>)
//  4. Removes unwanted elements (scripts, styles, navigation, etc.)
//  5. Converts cleaned HTML to Markdown
//  6. Post-processes Markdown (removes excess whitespace)
//  7. Adds YAML frontmatter with title, source URL, and fetch timestamp
//  8. Writes the final Markdown to the output file
//
// Parameters:
//   - htmlPath: Path to the HTML file to convert
//   - outputPath: Path where the Markdown file will be written
//   - sourceURL: Original URL where the HTML was fetched from (included in frontmatter)
//   - fetchedAt: ISO 8601 timestamp when the page was fetched (included in frontmatter)
//
// Returns an error if the HTML file cannot be read, parsed, or if the output file cannot be written.
// Logs a warning and returns nil if no main content is found in the HTML.
func (c *Converter) ConvertFile(htmlPath, outputPath, sourceURL, fetchedAt string) error {
	// Read HTML file
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to read HTML file: %w", err)
	}

	// Decode HTML with proper charset handling
	htmlString := decodeHTML(htmlContent)

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString))
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract title
	title := "Untitled"
	if titleText := doc.Find("title").First().Text(); titleText != "" {
		title = strings.TrimSpace(titleText)
	} else if h1Text := doc.Find("h1").First().Text(); h1Text != "" {
		title = strings.TrimSpace(h1Text)
	}

	// Extract main content
	var mainContent *goquery.Selection
	if main := doc.Find("main").First(); main.Length() > 0 {
		mainContent = main
	} else if article := doc.Find("article").First(); article.Length() > 0 {
		mainContent = article
	} else if content := doc.Find("div.content").First(); content.Length() > 0 {
		mainContent = content
	} else if body := doc.Find("body").First(); body.Length() > 0 {
		mainContent = body
	}

	if mainContent == nil || mainContent.Length() == 0 {
		log.Printf("Warning: No main content found in %s", htmlPath)
		return nil
	}

	// Clean HTML
	c.cleanHTML(mainContent)

	// Convert to Markdown
	mainHTML, err := mainContent.Html()
	if err != nil {
		return fmt.Errorf("failed to get HTML: %w", err)
	}

	markdown, err := c.mdConverter.ConvertString(mainHTML)
	if err != nil {
		return fmt.Errorf("failed to convert to markdown: %w", err)
	}

	// Post-process markdown
	markdown = c.postProcessMarkdown(markdown)

	// Create frontmatter
	escapedTitle := strings.ReplaceAll(title, `"`, `\"`)
	frontmatter := fmt.Sprintf(`---
title: "%s"
source_url: "%s"
fetched_at: "%s"
---

`, escapedTitle, sourceURL, fetchedAt)

	finalMD := frontmatter + markdown

	// Ensure output directory exists when one is specified
	outputDir := filepath.Dir(outputPath)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Write output
	if err := os.WriteFile(outputPath, []byte(finalMD), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	log.Printf("Converted: %s -> %s", htmlPath, outputPath)
	return nil
}

// cleanHTML removes unwanted elements from the selected HTML content.
// It eliminates scripts, styles, navigation elements, and other non-content sections
// to isolate the main documentation content.
func (c *Converter) cleanHTML(sel *goquery.Selection) {
	// Remove unwanted elements
	unwantedSelectors := []string{
		"script", "style", "meta", "link", "noscript", "iframe", "svg",
		".sidebar", "header", "footer", ".nav", ".menu", "#sidebar",
		".navigation", ".toc", "#toc", ".footer", "#footer",
	}

	for _, selector := range unwantedSelectors {
		sel.Find(selector).Remove()
	}
}

// postProcessMarkdown applies final formatting to Markdown content.
// It removes excessive blank lines and trailing whitespace from each line.
func (c *Converter) postProcessMarkdown(md string) string {
	// Remove multiple consecutive blank lines
	re := regexp.MustCompile(`\n{3,}`)
	md = re.ReplaceAllString(md, "\n\n")

	// Remove trailing whitespace from each line
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	return strings.Join(lines, "\n")
}

// decodeHTML decodes HTML bytes to a UTF-8 string using character encoding detection.
// It attempts to detect the character encoding from HTML meta tags (both <meta charset>
// and <meta http-equiv="Content-Type"> formats) and decodes accordingly.
//
// If no encoding is detected or decoding fails, it falls back to treating the input as UTF-8.
// This ensures proper handling of non-UTF-8 HTML documents (e.g., Shift-JIS, EUC-JP).
//
// Parameters:
//   - body: The raw HTML bytes to decode
//
// Returns a UTF-8 string representation of the HTML content.
func decodeHTML(body []byte) string {
	// Try to get charset from HTML meta tag
	enc := getEncodingFromMeta(body)
	if enc != nil {
		decoded, err := decodeWithEncoding(body, enc)
		if err == nil {
			return decoded
		}
	}

	// Fallback to UTF-8
	return string(body)
}

// getEncodingFromMeta extracts the character encoding from HTML meta tags using regex on raw bytes.
// This approach avoids parsing HTML with an incorrect encoding, which would corrupt the content.
// It supports both <meta charset="..."> and <meta http-equiv="Content-Type"> formats.
func getEncodingFromMeta(body []byte) encoding.Encoding {
	// Quick check for BOM first
	if len(body) >= 3 && body[0] == 0xEF && body[1] == 0xBB && body[2] == 0xBF {
		// UTF-8 BOM
		return nil // UTF-8 is default
	}

	// Use regex to find charset in raw bytes (works for ASCII-compatible encodings)
	// Pattern 1: <meta charset="...">
	charsetRe := regexp.MustCompile(`(?i)<meta[^>]+charset=["']?([^"'\s>]+)`)
	if matches := charsetRe.Find(body); matches != nil {
		submatches := charsetRe.FindSubmatch(body)
		if len(submatches) > 1 {
			charset := string(submatches[1])
			if enc, err := htmlindex.Get(charset); err == nil {
				return enc
			}
		}
	}

	// Pattern 2: <meta http-equiv="Content-Type" content="text/html; charset=...">
	contentTypeRe := regexp.MustCompile(`(?i)<meta[^>]+http-equiv=["']?Content-Type["']?[^>]+content=["']?[^"']*charset=([^"'\s;>]+)`)
	if matches := contentTypeRe.Find(body); matches != nil {
		submatches := contentTypeRe.FindSubmatch(body)
		if len(submatches) > 1 {
			charset := string(submatches[1])
			if enc, err := htmlindex.Get(charset); err == nil {
				return enc
			}
		}
	}

	// Pattern 3: content attribute before http-equiv (reverse order)
	contentTypeRe2 := regexp.MustCompile(`(?i)<meta[^>]+content=["']?[^"']*charset=([^"'\s;>]+)[^>]+http-equiv=["']?Content-Type["']?`)
	if matches := contentTypeRe2.Find(body); matches != nil {
		submatches := contentTypeRe2.FindSubmatch(body)
		if len(submatches) > 1 {
			charset := string(submatches[1])
			if enc, err := htmlindex.Get(charset); err == nil {
				return enc
			}
		}
	}

	return nil
}

// decodeWithEncoding decodes bytes from a specific character encoding to UTF-8.
// It uses the golang.org/x/text/transform package to perform the conversion.
//
// Parameters:
//   - body: The bytes to decode
//   - enc: The source character encoding (e.g., Shift-JIS, ISO-8859-1)
//
// Returns the decoded UTF-8 string, or an error if decoding fails.
func decodeWithEncoding(body []byte, enc encoding.Encoding) (string, error) {
	reader := transform.NewReader(bytes.NewReader(body), enc.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
