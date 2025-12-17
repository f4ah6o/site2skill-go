// Package packager provides skill packaging functionality.
// It creates .skill files (ZIP archives) from a skill directory structure,
// preserving the directory hierarchy and file metadata.
package packager

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Packager creates .skill files (ZIP archives) from skill directories.
// .skill files are ZIP archives containing the complete skill structure:
// SKILL.md manifest, documentation files, and optional scripts. These archives
// can be distributed and installed for use with Claude or Codex.
type Packager struct{}

// New creates a new Packager instance.
//
// Returns a Packager ready to create .skill archives.
func New() *Packager {
	return &Packager{}
}

// Package creates a .skill file (ZIP archive) from a skill directory.
// The entire skill directory structure is compressed into a single .skill file,
// which is a ZIP archive with a .skill extension for easy distribution.
//
// The archive structure preserves the original directory layout:
//   skillname.skill (ZIP archive containing:)
//     ├── SKILL.md
//     └── docs/
//         ├── file1.md
//         └── file2.md
//
// All files are compressed using DEFLATE compression. Directory entries are included
// in the archive to preserve empty directories if present.
//
// Parameters:
//   - skillDir: Path to the skill directory to package (e.g., ".claude/skills/myskill")
//   - outputDir: Directory where the .skill file will be created (e.g., ".claude/skills")
//
// Returns:
//   - The full path to the created .skill file (e.g., ".claude/skills/myskill.skill")
//   - An error if the skill directory doesn't exist, cannot be read, or the archive cannot be created
//
// The .skill file is named after the skill directory's basename (e.g., "myskill" -> "myskill.skill").
func (p *Packager) Package(skillDir, outputDir string) (string, error) {
	// Check if skill directory exists
	if info, err := os.Stat(skillDir); os.IsNotExist(err) || !info.IsDir() {
		return "", fmt.Errorf("directory not found: %s", skillDir)
	}

	skillName := filepath.Base(skillDir)
	outputFilename := filepath.Join(outputDir, skillName+".skill")

	log.Printf("Packaging %s to %s...", skillDir, outputFilename)

	// Create zip file
	zipFile, err := os.Create(outputFilename)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through skill directory and add files to zip
	err = filepath.Walk(skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create zip header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Use forward slashes for zip paths (cross-platform compatibility)
		header.Name = strings.ReplaceAll(relPath, string(os.PathSeparator), "/")

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to package skill: %w", err)
	}

	log.Printf("Successfully created: %s", outputFilename)
	return outputFilename, nil
}
