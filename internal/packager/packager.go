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

// Packager creates .skill files from skill directories.
type Packager struct{}

// New creates a new Packager instance.
func New() *Packager {
	return &Packager{}
}

// Package creates a .skill file (ZIP archive) from the skill directory.
// It recursively includes all files and directories from skillDir.
// skillDir: path to the skill directory to package
// outputDir: directory where the .skill file will be created
// Returns the path to the created .skill file or an error.
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
