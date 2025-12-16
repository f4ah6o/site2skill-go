package normalizer

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Normalizer struct{}

func New() *Normalizer {
	return &Normalizer{}
}

type Frontmatter struct {
	Title      string `yaml:"title"`
	SourceURL  string `yaml:"source_url"`
	FetchedAt  string `yaml:"fetched_at"`
}

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
