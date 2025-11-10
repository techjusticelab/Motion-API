//go:build enhanced
// +build enhanced

package main

import "motion-index-fiber/pkg/processing/extractor"

func setOCRMaxPages(cfg *extractor.EnhancedConfig, pages int) {
	if cfg == nil || cfg.OCRConfig == nil || pages <= 0 {
		return
	}

	cfg.OCRConfig.MaxPages = pages
}