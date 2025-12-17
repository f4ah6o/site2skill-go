// Package skillgen provides skill structure generation functionality.
// It creates skill directory layouts, generates SKILL.md manifests for different AI models,
// and copies documentation files and search scripts into the skill structure.
package skillgen

import (
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
	// FormatBoth specifies generation of both Claude and Codex-compatible skills.
	FormatBoth = "both"
)


// Generator creates skill directory structures and manifests for AI skill packages.
// It generates platform-specific skill layouts with appropriate documentation and metadata
// files for Claude or Codex AI assistants.
type Generator struct {
	// format specifies the target AI platform ("claude", "codex", or "both")
	format string
}

// New creates a new Generator configured for the specified output format.
//
// Parameters:
//   - format: The target format - "claude" for Claude Agent Skills,
//     "codex" for OpenAI Codex Skills, or "both" for generating both formats
//
// Returns a Generator ready to create skill directory structures.
func New(format string) *Generator {
	return &Generator{
		format: format,
	}
}

// Generate creates a complete skill directory structure for the specified skill package.
// It sets up the directory layout, generates platform-specific manifest files, and
// copies documentation files into the proper structure.
//
// The generated structure:
//   skillName/
//     ├── SKILL.md          # Platform-specific manifest and usage instructions
//     └── docs/             # Markdown documentation files with YAML frontmatter
//         ├── file1.md
//         ├── file2.md
//         └── ...
//
// Parameters:
//   - skillName: Name of the skill (used as the directory name)
//   - sourceDir: Directory containing source Markdown files to include in the skill
//   - outputBase: Base directory where the skill directory will be created
//
// Returns an error if directories cannot be created, the manifest cannot be written,
// or documentation files cannot be copied.
//
// Example:
//   gen := New("claude")
//   err := gen.Generate("python-docs", "./markdown", ".claude/skills")
//   // Creates: .claude/skills/python-docs/SKILL.md and .claude/skills/python-docs/docs/*.md
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

// createSkillMD generates the SKILL.md manifest file for the skill package.
// The manifest provides instructions for AI assistants on how to use the skill,
// including search commands, file locations, and response formatting guidelines.
//
// The content is customized for the target platform:
//   - Claude format: Includes YAML frontmatter with name and description
//   - Codex format: Uses standard Markdown headings without frontmatter
//
// Parameters:
//   - skillDir: The skill's root directory where SKILL.md will be created
//   - skillName: The skill name (used in the manifest content and metadata)
//
// Returns an error if the file cannot be written.
func (g *Generator) createSkillMD(skillDir, skillName string) error {
	skillMDPath := filepath.Join(skillDir, "SKILL.md")

	var content string
	if g.format == FormatCodex {
		content = g.getCodexSkillContent(skillName)
	} else if g.format == FormatBoth {
		// For "both" format, use Claude format by default (will be handled by Generator)
		content = g.getClaudeSkillContent(skillName)
	} else {
		content = g.getClaudeSkillContent(skillName)
	}

	if err := os.WriteFile(skillMDPath, []byte(content), 0644); err != nil {
		return err
	}

	log.Printf("Created %s", skillMDPath)
	return nil
}

// getClaudeSkillContent generates the SKILL.md manifest content for Claude Agent Skills.
// The manifest includes YAML frontmatter and detailed instructions for Claude on how to
// search documentation, format responses, and cite sources.
//
// Parameters:
//   - skillName: The skill name (used in metadata and content)
//
// Returns a string containing the complete SKILL.md content formatted for Claude.
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

Use the `+"`site2skillgo search`"+` command to search through documentation:

`+"```bash"+`
site2skillgo search "<query>" --skill-dir .
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

// getCodexSkillContent generates the SKILL.md manifest content for OpenAI Codex Skills.
// The manifest uses standard Markdown headings and provides instructions for Codex on
// searching documentation, understanding file structure, and best practices.
//
// Parameters:
//   - skillName: The skill name (used in the content)
//
// Returns a string containing the complete SKILL.md content formatted for Codex.
func (g *Generator) getCodexSkillContent(skillName string) string {
	return fmt.Sprintf(`# %s Documentation Skill

This skill provides access to %s documentation for OpenAI Codex.

## Structure

- `+"`docs/`"+`: Contains all documentation as Markdown files

## Search Documentation

Use the `+"`site2skillgo search`"+` command to find relevant documentation:

`+"```bash"+`
site2skillgo search "your query here" --skill-dir .
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
site2skillgo search "authentication api key" --skill-dir .

# Get top 5 results as JSON
site2skillgo search "payment methods" --json --max-results 5 --skill-dir .
`+"```"+`
`, strings.ToUpper(skillName), strings.ToUpper(skillName))
}

// copyMarkdownFiles copies all Markdown files from the source directory to the skill's docs directory.
// It recursively walks the source directory and copies only .md files, flattening the structure
// (all files go directly into docs/ regardless of source subdirectories).
//
// Security: Performs path validation to prevent directory traversal attacks by checking that
// destination paths remain within the docs directory.
//
// Parameters:
//   - sourceDir: Source directory containing Markdown files to copy
//   - docsDir: Destination docs/ directory within the skill
//
// Returns an error if the source directory doesn't exist, files cannot be read, or the
// destination cannot be written. Logs warnings for skipped files and info about copy progress.
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
