// Package validator provides skill validation functionality.
// It checks skill directory structure, validates required files like SKILL.md,
// ensures documentation is present, and analyzes skill size against platform limits.
package validator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Validator validates skill directory structures and content.
type Validator struct{}

// New creates a new Validator instance.
func New() *Validator {
	return &Validator{}
}

// Validate checks that a skill directory has the required structure and content.
// It verifies the presence of SKILL.md, docs/ directory with Markdown files,
// validates YAML frontmatter, and analyzes skill size.
// skillDir: path to the skill directory to validate
// Returns true if validation passes, false otherwise.
func (v *Validator) Validate(skillDir string) bool {
	log.Printf("Validating skill in: %s", skillDir)

	var errors []string
	var warnings []string

	// 1. Check directory existence
	if info, err := os.Stat(skillDir); os.IsNotExist(err) || !info.IsDir() {
		log.Printf("Error: Directory not found: %s", skillDir)
		return false
	}

	// 2. Check SKILL.md
	skillMDPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
		errors = append(errors, "SKILL.md not found.")
	} else {
		log.Printf("Found SKILL.md")
		// Validate frontmatter
		content, err := os.ReadFile(skillMDPath)
		if err == nil {
			if strings.HasPrefix(string(content), "---\n") {
				re := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
				matches := re.FindStringSubmatch(string(content))
				if len(matches) > 1 {
					frontmatter := matches[1]
					requiredFields := []string{"name", "description"}
					for _, field := range requiredFields {
						if !strings.Contains(frontmatter, field+":") {
							warnings = append(warnings, fmt.Sprintf("SKILL.md frontmatter missing '%s' field", field))
						}
					}
					log.Printf("  YAML frontmatter present")
				} else {
					warnings = append(warnings, "SKILL.md has incomplete frontmatter")
				}
			} else {
				warnings = append(warnings, "SKILL.md missing YAML frontmatter")
			}
		}
	}

	// 3. Check docs/ directory
	docsDir := filepath.Join(skillDir, "docs")
	if info, err := os.Stat(docsDir); os.IsNotExist(err) || !info.IsDir() {
		errors = append(errors, "docs/ directory not found.")
	} else {
		log.Printf("Found docs/")

		// Count markdown files
		mdFiles := []string{}
		filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && filepath.Ext(path) == ".md" {
				mdFiles = append(mdFiles, path)
			}
			return nil
		})

		if len(mdFiles) == 0 {
			warnings = append(warnings, "docs/ directory is empty (no .md files)")
		} else {
			log.Printf("  %d markdown files", len(mdFiles))
		}
	}

	// 4. Check optional directories
	scriptsDir := filepath.Join(skillDir, "scripts")
	if info, err := os.Stat(scriptsDir); err == nil && info.IsDir() {
		log.Printf("Found scripts/ (optional)")
	}

	// 5. Check skill size
	v.checkSkillSize(skillDir)

	// 6. Report results
	if len(errors) > 0 {
		log.Printf("VALIDATION FAILED:")
		for _, err := range errors {
			log.Printf("  - %s", err)
		}
		return false
	}

	if len(warnings) > 0 {
		log.Printf("Warnings:")
		for _, warn := range warnings {
			log.Printf("  - %s", warn)
		}
	}

	log.Printf("Validation passed!")
	return true
}

// fileSize represents the size and path of a file.
type fileSize struct {
	// size is the file size in bytes.
	size int64
	// path is the file path.
	path string
}

func (v *Validator) checkSkillSize(skillDir string) {
	docsDir := filepath.Join(skillDir, "docs")
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		return
	}

	var totalSize int64
	var fileSizes []fileSize

	filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size := info.Size()
			totalSize += size
			fileSizes = append(fileSizes, fileSize{size: size, path: path})
		}
		return nil
	})

	// Sort by size descending
	sort.Slice(fileSizes, func(i, j int) bool {
		return fileSizes[i].size > fileSizes[j].size
	})

	totalSizeMB := float64(totalSize) / (1024 * 1024)
	log.Printf("\n--- Skill Size Analysis ---")
	log.Printf("Total Uncompressed Size: %.2f MB", totalSizeMB)

	if totalSize > 8*1024*1024 {
		log.Printf("Warning: Skill uncompressed size exceeds Claude's 8MB limit.")
		log.Printf("Warning: The skill may fail to load in Claude.")
	} else {
		log.Printf("Size is within limits (< 8MB).")
	}

	log.Printf("\nTop 10 Largest Files:")
	for i := 0; i < 10 && i < len(fileSizes); i++ {
		relPath, _ := filepath.Rel(skillDir, fileSizes[i].path)
		sizeKB := float64(fileSizes[i].size) / 1024
		log.Printf("  %.1f KB - %s", sizeKB, relPath)
	}
	log.Printf("---------------------------\n")
}
