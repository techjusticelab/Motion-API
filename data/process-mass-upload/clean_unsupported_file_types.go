//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Set default directory to current directory if none specified
	startDir := "."
	if len(os.Args) > 1 {
		startDir = os.Args[1]
	}

	fmt.Printf("Starting deletion of non-PDF convertible files in: %s\n", startDir)
	fmt.Println("-------------------------------------------------------")

	// List of file extensions that should be kept
	convertibleExts := map[string]bool{
		"pdf":  true,
		"doc":  true,
		"docx": true,
		"txt":  true,
		"ppt":  true,
		"pptx": true,
	}

	// Walk through all files and directories
	err := filepath.Walk(startDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %s: %v\n", path, err)
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get file extension
		ext := strings.ToLower(filepath.Ext(path))
		if ext == "" {
			return nil // Skip files with no extension
		}

		// Remove leading dot from extension
		ext = strings.TrimPrefix(ext, ".")

		// Check if extension is in our keep list
		if !convertibleExts[ext] {
			fmt.Printf("Deleting: %s (.%s)\n", path, ext)
			err := os.Remove(path)
			if err != nil {
				fmt.Printf("Error deleting %s: %v\n", path, err)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path: %v\n", err)
	}

	fmt.Println("-------------------------------------------------------")
	fmt.Println("Deletion complete!")
}
