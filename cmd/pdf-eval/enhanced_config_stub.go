//go:build !enhanced
// +build !enhanced

package main

import "motion-index-fiber/pkg/processing/extractor"

func setOCRMaxPages(cfg *extractor.EnhancedConfig, pages int) {}