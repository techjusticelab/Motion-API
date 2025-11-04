//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// normalizeFilename removes common separators and converts to lowercase
func normalizeFilename(filename string) string {
	// Remove extension
	basename := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Replace hyphens, underscores, and multiple spaces with single space
	re := regexp.MustCompile(`[-_]+`)
	normalized := re.ReplaceAllString(basename, " ")

	// Remove any parentheses and their contents
	re = regexp.MustCompile(`\s*\([^)]*\)\s*`)
	normalized = re.ReplaceAllString(normalized, " ")

	// Collapse multiple spaces into one
	re = regexp.MustCompile(`\s+`)
	normalized = re.ReplaceAllString(normalized, " ")

	// Trim spaces and convert to lowercase
	normalized = strings.TrimSpace(normalized)
	normalized = strings.ToLower(normalized)

	return normalized
}

func main() {
	// Map to track filenames without extensions
	fileMap := make(map[string][]string)

	// Walk through all directories recursively
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get the extension
		ext := strings.ToLower(filepath.Ext(path))
		if ext == "" {
			return nil
		}

		// Get normalized filename without extension
		baseNoExt := normalizeFilename(filepath.Base(path))
		if baseNoExt == "" {
			return nil
		}

		// Add to our map
		fileMap[baseNoExt] = append(fileMap[baseNoExt], path)

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path: %v\n", err)
		os.Exit(1)
	}

	// Process duplicate files
	for normalizedName, files := range fileMap {
		if len(files) <= 1 {
			continue // Skip if there's only one file with this normalized name
		}

		fmt.Printf("\nFound potential duplicates for '%s':\n", normalizedName)
		for _, f := range files {
			fmt.Printf("  - %s\n", f)
		}

		var pdfFile string
		var otherFiles []string

		// Find PDF file if it exists
		for _, file := range files {
			ext := strings.ToLower(filepath.Ext(file))
			if ext == ".pdf" {
				pdfFile = file
			} else {
				otherFiles = append(otherFiles, file)
			}
		}

		// If PDF exists, delete the other files
		if pdfFile != "" {
			fmt.Printf("Keeping PDF: %s\n", pdfFile)
			for _, file := range otherFiles {
				fmt.Printf("Deleting: %s\n", file)
				err := os.Remove(file)
				if err != nil {
					fmt.Printf("Error deleting %s: %v\n", file, err)
				}
			}
		} else {
			fmt.Printf("No PDF version found for this group, keeping all files.\n")
		}
	}

	fmt.Println("\nProcess completed.")
}
