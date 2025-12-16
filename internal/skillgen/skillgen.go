// Package skillgen provides skill structure generation functionality.
// It creates skill directory layouts, generates SKILL.md manifests for different AI models,
// and copies documentation files and search scripts into the skill structure.
package skillgen

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// FormatClaude specifies generation of a Claude-compatible skill.
	FormatClaude = "claude"
	// FormatCodex specifies generation of an OpenAI Codex-compatible skill.
	FormatCodex = "codex"
)

// templates embeds template files for SKILL.md and search scripts.
//go:embed templates/*
var templates embed.FS

// Generator creates skill directory structures.
type Generator struct {
	format string
}

// New creates a new Generator configured for the specified format (claude or codex).
func New(format string) *Generator {
	return &Generator{
		format: format,
	}
}

// Generate creates a complete skill directory structure for the specified skill.
// It creates docs/ and scripts/ directories, generates an appropriate SKILL.md manifest,
// copies search scripts, and copies markdown documentation files.
// skillName: name of the skill to generate
// sourceDir: directory containing Markdown documentation files
// outputBase: base output directory where the skill directory will be created
func (g *Generator) Generate(skillName, sourceDir, outputBase string) error {
	skillDir := filepath.Join(outputBase, skillName)
	docsDir := filepath.Join(skillDir, "docs")

	// Create directories
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	// Create SKILL.md based on format
	if err := g.createSkillMD(skillDir, skillName); err != nil {
		return fmt.Errorf("failed to create SKILL.md: %w", err)
	}

	// Copy markdown files
	if err := g.copyMarkdownFiles(sourceDir, docsDir); err != nil {
		return fmt.Errorf("failed to copy markdown files: %w", err)
	}

	return nil
}

func (g *Generator) createSkillMD(skillDir, skillName string) error {
	skillMDPath := filepath.Join(skillDir, "SKILL.md")

	var content string
	if g.format == FormatCodex {
		content = g.getCodexSkillContent(skillName)
	} else {
		content = g.getClaudeSkillContent(skillName)
	}

	if err := os.WriteFile(skillMDPath, []byte(content), 0644); err != nil {
		return err
	}

	log.Printf("Created %s", skillMDPath)
	return nil
}

func (g *Generator) getClaudeSkillContent(skillName string) string {
	return fmt.Sprintf(`---
name: %s
description: %s documentation assistant
---

# %s Skill

This skill provides access to %s documentation.

## Documentation

All documentation files are in the `+"`docs/`"+` directory as Markdown files.

## Search Tool

Use the `+"`s2s-go search`"+` command to search through documentation:

`+"```bash"+`
s2s-go search "<query>" --skill-dir .
`+"```"+`

Options:
- `+"`--json`"+` - Output as JSON
- `+"`--max-results N`"+` - Limit results (default: 10)
- `+"`--skill-dir PATH`"+` - Path to skill directory (default: current directory)

## Usage

1. Search or read files in `+"`docs/`"+` for relevant information
2. Each file has frontmatter with `+"`source_url`"+` and `+"`fetched_at`"+`
3. Always cite the source URL in responses
4. Note the fetch date - documentation may have changed

## Response Format

`+"```"+`
[Answer based on documentation]

**Source:** [source_url]
**Fetched:** [fetched_at]
`+"```"+`
`, skillName, strings.ToUpper(skillName), strings.ToUpper(skillName), strings.ToUpper(skillName))
}

func (g *Generator) getCodexSkillContent(skillName string) string {
	return fmt.Sprintf(`# %s Documentation Skill

This skill provides access to %s documentation for OpenAI Codex.

## Structure

- `+"`docs/`"+`: Contains all documentation as Markdown files

## Search Documentation

Use the `+"`s2s-go search`"+` command to find relevant documentation:

`+"```bash"+`
s2s-go search "your query here" --skill-dir .
`+"```"+`

Options:
- `+"`--json`"+`: Output results as JSON
- `+"`--max-results N`"+`: Limit number of results (default: 10)
- `+"`--skill-dir PATH`"+`: Path to skill directory (default: current directory)

## Documentation Files

Each file in `+"`docs/`"+` contains:
- **Frontmatter**: YAML metadata with `+"`title`"+`, `+"`source_url`"+`, and `+"`fetched_at`"+`
- **Content**: Markdown-formatted documentation

## Best Practices

1. Search for relevant topics using the search script
2. Read the full documentation file for context
3. Always reference the source URL when providing information
4. Note the fetch date as documentation may have been updated

## Example Usage

`+"```bash"+`
# Search for authentication documentation
s2s-go search "authentication api key" --skill-dir .

# Get top 5 results as JSON
s2s-go search "payment methods" --json --max-results 5 --skill-dir .
`+"```"+`
`, strings.ToUpper(skillName), strings.ToUpper(skillName))
}

func (g *Generator) copySearchScript(scriptsDir string) error {
	var templateName string
	if g.format == FormatCodex {
		templateName = "templates/search_docs_codex.py"
	} else {
		templateName = "templates/search_docs_claude.py"
	}

	content, err := templates.ReadFile(templateName)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	destPath := filepath.Join(scriptsDir, "search_docs.py")
	if err := os.WriteFile(destPath, content, 0755); err != nil {
		return err
	}

	log.Printf("Installed search_docs.py (%s format)", g.format)
	return nil
}

func (g *Generator) copyMarkdownFiles(sourceDir, docsDir string) error {
	if sourceDir == "" {
		return fmt.Errorf("source directory is empty")
	}

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", sourceDir)
	}

	fileCount := 0
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".md" {
			fileName := filepath.Base(path)
			dstPath := filepath.Join(docsDir, fileName)

			// Security check
			absDstPath, err := filepath.Abs(dstPath)
			if err != nil {
				return err
			}
			absDocsDir, err := filepath.Abs(docsDir)
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(absDocsDir, absDstPath)
			if err != nil || strings.HasPrefix(relPath, "..") {
				log.Printf("Warning: skipping potential path traversal file: %s", fileName)
				return nil
			}

			// Copy file
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", path, err)
			}

			if err := os.WriteFile(dstPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", dstPath, err)
			}

			fileCount++
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("Copied %d files to docs/", fileCount)
	return nil
}
