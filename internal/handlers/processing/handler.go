package processing

import (
	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/processing/pipeline"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// Handler provides document processing HTTP handlers (legacy implementation)
type Handler struct {
	cfg       *config.Config
	pipeline  pipeline.Pipeline
	storage   storage.Service
	searchSvc search.Service
}

// NewHandler creates a new processing handler
func NewHandler(cfg *config.Config, pipeline pipeline.Pipeline, storage storage.Service, searchSvc search.Service) *Handler {
	return &Handler{
		cfg:       cfg,
		pipeline:  pipeline,
		storage:   storage,
		searchSvc: searchSvc,
	}
}
