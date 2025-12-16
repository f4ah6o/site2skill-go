// Package converter provides HTML to Markdown conversion functionality.
// It extracts main content from HTML pages, cleans unwanted elements,
// and converts to Markdown with YAML frontmatter containing metadata.
package converter

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/JohannesKaufmann/html-to-markdown"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// Converter converts HTML content to Markdown with YAML frontmatter.
type Converter struct {
	mdConverter *md.Converter
}

// New creates a new Converter instance.
func New() *Converter {
	converter := md.NewConverter("", true, nil)
	return &Converter{
		mdConverter: converter,
	}
}

// ConvertFile converts an HTML file to Markdown with frontmatter.
// It extracts the main content, removes unwanted elements, and adds YAML frontmatter
// with the document title, source URL, and fetch timestamp.
// htmlPath: path to the HTML file to convert
// outputPath: path where the Markdown file will be written
// sourceURL: URL where the HTML was fetched from
// fetchedAt: ISO 3339 timestamp when the page was fetched
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

	// Ensure output directory exists
	if err := os.MkdirAll(strings.TrimSuffix(outputPath, "/"+strings.Split(outputPath, "/")[len(strings.Split(outputPath, "/"))-1]), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write output
	if err := os.WriteFile(outputPath, []byte(finalMD), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	log.Printf("Converted: %s -> %s", htmlPath, outputPath)
	return nil
}

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

// decodeHTML decodes HTML bytes to string using detected charset.
// It tries to extract charset from HTML meta tag.
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

// getEncodingFromMeta extracts charset from HTML meta tag.
func getEncodingFromMeta(body []byte) encoding.Encoding {
	// Quick check for BOM first
	if len(body) >= 3 && body[0] == 0xEF && body[1] == 0xBB && body[2] == 0xBF {
		// UTF-8 BOM
		return nil // UTF-8 is default
	}

	// Parse HTML to find meta charset
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil
	}

	var findMeta func(*html.Node) encoding.Encoding
	findMeta = func(n *html.Node) encoding.Encoding {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var charset string
			for _, attr := range n.Attr {
				if attr.Key == "charset" {
					charset = attr.Val
					break
				}
				if attr.Key == "http-equiv" && strings.ToLower(attr.Val) == "content-type" {
					// Look for content attribute
					for _, a := range n.Attr {
						if a.Key == "content" {
							re := regexp.MustCompile(`charset=([^\s;]+)`)
							matches := re.FindStringSubmatch(a.Val)
							if len(matches) > 1 {
								charset = strings.Trim(matches[1], `"'`)
							}
							break
						}
					}
				}
			}
			if charset != "" {
				enc, err := htmlindex.Get(charset)
				if err == nil {
					return enc
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if enc := findMeta(c); enc != nil {
				return enc
			}
		}
		return nil
	}

	return findMeta(doc)
}

// decodeWithEncoding decodes bytes using specified encoding.
func decodeWithEncoding(body []byte, enc encoding.Encoding) (string, error) {
	reader := transform.NewReader(strings.NewReader(string(body)), enc.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
