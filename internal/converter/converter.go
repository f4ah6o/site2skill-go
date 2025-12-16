package converter

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/JohannesKaufmann/html-to-markdown"
)

type Converter struct {
	mdConverter *md.Converter
}

func New() *Converter {
	converter := md.NewConverter("", true, nil)
	return &Converter{
		mdConverter: converter,
	}
}

func (c *Converter) ConvertFile(htmlPath, outputPath, sourceURL, fetchedAt string) error {
	// Read HTML file
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to read HTML file: %w", err)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
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
