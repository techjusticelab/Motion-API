package extractor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// convertWithLibreOffice uses LibreOffice in headless mode to convert supported office documents
// to plain text. The extension parameter should be provided with or without a leading dot (e.g. "doc" or ".ppt").
func convertWithLibreOffice(ctx context.Context, content []byte, extension string) (string, error) {
	ext := strings.TrimPrefix(strings.ToLower(extension), ".")
	if ext == "" {
		return "", fmt.Errorf("missing file extension for LibreOffice conversion")
	}

	tempDir, err := os.MkdirTemp("", "office-extract-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	inputName := fmt.Sprintf("document.%s", ext)
	inputPath := filepath.Join(tempDir, inputName)
	if err := os.WriteFile(inputPath, content, 0o600); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}

	profileDir, err := os.MkdirTemp(tempDir, "profile-*")
	if err != nil {
		return "", fmt.Errorf("create temp profile dir: %w", err)
	}

	profileURI := "file://" + filepath.ToSlash(profileDir)

	args := []string{
		"--headless",
		"--norestore",
		"--nolockcheck",
		"--nodefault",
		fmt.Sprintf("-env:UserInstallation=%s", profileURI),
		"--convert-to", "txt:Text",
		"--outdir", tempDir,
		inputPath,
	}

	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, "soffice", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("libreoffice conversion failed: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	var outputPath string
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return "", fmt.Errorf("list converted files: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") {
			outputPath = filepath.Join(tempDir, entry.Name())
			break
		}
	}

	if outputPath == "" {
		return "", fmt.Errorf("converted text file not found")
	}

	textBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("read converted text: %w", err)
	}

	return string(textBytes), nil
}
