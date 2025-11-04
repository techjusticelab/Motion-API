//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
)

var supportedExtensions = map[string]struct{}{
	".doc":  {},
	".docx": {},
	".pdf":  {},
	".ppt":  {},
	".pptx": {},
	".txt":  {},
}

type uploadJob struct {
	path    string
	display string
}

type uploadResult struct {
	job      uploadJob
	attempts int
	err      error
}

type fileCandidate struct {
	path string
	size int64
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "clean":
		fmt.Println("Running clean_unsupported_file_types.go functionality...")
		fmt.Println("Please run: go run clean_unsupported_file_types.go [directory]")

	case "dedupe":
		fmt.Println("Running delete_duplicate_files.go functionality...")
		fmt.Println("Please run: go run delete_duplicate_files.go [directory]")

	case "upload":
		if err := runUpload(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}
