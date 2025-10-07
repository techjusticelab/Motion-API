package processing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/processing/queue"
)

// IndexingQueueItem represents an item in the indexing queue
type IndexingQueueItem struct {
	DocumentID           string                           `json:"document_id"`
	DocumentPath         string                           `json:"document_path,omitempty"`
	Text                 string                           `json:"text"`
	ClassificationResult *classifier.ClassificationResult `json:"classification_result"`
	FileName             string                           `json:"file_name,omitempty"`
	ContentType          string                           `json:"content_type,omitempty"`
	Size                 int64                            `json:"size,omitempty"`
	FileURL              string                           `json:"file_url,omitempty"`
	RetryCount           int                              `json:"retry_count"`
	SourceJobID          string                           `json:"source_job_id,omitempty"`
}

// IndexingProcessorConfig holds configuration for the indexing processor
type IndexingProcessorConfig struct {
	APIBaseURL     string        `json:"api_base_url"`
	RequestTimeout time.Duration `json:"request_timeout"`
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// IndexingProcessor handles processing of indexing queue items
type IndexingProcessor struct {
	config     *IndexingProcessorConfig
	httpClient *http.Client
}

// NewIndexingProcessor creates a new indexing processor
func NewIndexingProcessor(config *IndexingProcessorConfig) *IndexingProcessor {
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 120 * time.Second // Increased from 30s to 120s for large documents
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}
	if config.APIBaseURL == "" {
		config.APIBaseURL = "http://localhost:6000"
	}

	return &IndexingProcessor{
		config: config,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
		},
	}
}

// ProcessIndexingItem is the queue processor function for indexing items
func (p *IndexingProcessor) ProcessIndexingItem(ctx context.Context, item *queue.QueueItem) *queue.ProcessingResult {
	startTime := time.Now()

	// Parse the queue item data
	indexingItem, err := p.parseQueueItem(item)
	if err != nil {
		log.Printf("[INDEXING] Failed to parse queue item %s: %v", item.ID, err)
		return &queue.ProcessingResult{
			Success:     false,
			Error:       fmt.Errorf("failed to parse queue item: %w", err),
			Duration:    time.Since(startTime),
			ShouldRetry: false, // Don't retry parse errors
		}
	}

	// Attempt to index the document
	indexResult, err := p.indexDocument(ctx, indexingItem)
	if err != nil {
		log.Printf("[INDEXING] Failed to index document %s (attempt %d/%d): %v", 
			indexingItem.DocumentID, item.RetryCount+1, p.config.MaxRetries, err)
		
		shouldRetry := item.RetryCount < p.config.MaxRetries
		return &queue.ProcessingResult{
			Success:     false,
			Error:       err,
			Duration:    time.Since(startTime),
			ShouldRetry: shouldRetry,
		}
	}

	log.Printf("[INDEXING] Successfully indexed document %s with ID: %s", 
		indexingItem.DocumentID, indexResult.IndexID)

	return &queue.ProcessingResult{
		Success:     true,
		Duration:    time.Since(startTime),
		Output:      indexResult,
		ShouldRetry: false,
		Metadata: map[string]interface{}{
			"document_id": indexingItem.DocumentID,
			"index_id":    indexResult.IndexID,
			"indexed_at":  indexResult.IndexedAt,
		},
	}
}

// parseQueueItem parses the queue item data into an IndexingQueueItem
func (p *IndexingProcessor) parseQueueItem(item *queue.QueueItem) (*IndexingQueueItem, error) {
	if item.Type != queue.QueueTypeIndexing {
		return nil, fmt.Errorf("invalid queue item type: %s", item.Type)
	}

	// Try to parse as IndexingQueueItem
	var indexingItem IndexingQueueItem
	
	// Handle different data formats
	switch data := item.Data.(type) {
	case map[string]interface{}:
		// Convert from generic map
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal item data: %w", err)
		}
		if err := json.Unmarshal(jsonData, &indexingItem); err != nil {
			return nil, fmt.Errorf("failed to unmarshal item data: %w", err)
		}
	case *IndexingQueueItem:
		indexingItem = *data
	case IndexingQueueItem:
		indexingItem = data
	default:
		return nil, fmt.Errorf("unsupported data type: %T", data)
	}

	// Validate required fields
	if indexingItem.DocumentID == "" {
		return nil, fmt.Errorf("document_id is required")
	}
	if indexingItem.Text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if indexingItem.ClassificationResult == nil {
		return nil, fmt.Errorf("classification_result is required")
	}

	return &indexingItem, nil
}

// indexDocument calls the indexing API endpoint
func (p *IndexingProcessor) indexDocument(ctx context.Context, item *IndexingQueueItem) (*models.IndexDocumentResponse, error) {
	// Prepare the API request
	apiRequest := &models.IndexDocumentRequest{
		DocumentID:           item.DocumentID,
		DocumentPath:         item.DocumentPath,
		Text:                 item.Text,
		ClassificationResult: item.ClassificationResult,
		FileName:             item.FileName,
		ContentType:          item.ContentType,
		Size:                 item.Size,
		FileURL:              item.FileURL,
	}

	// Serialize request to JSON
	requestBody, err := json.Marshal(apiRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/index/document", p.config.APIBaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("indexing failed with status %d", resp.StatusCode)
	}

	// Parse response
	var apiResponse struct {
		Success   bool                             `json:"success"`
		Data      *models.IndexDocumentResponse    `json:"data"`
		Message   string                           `json:"message"`
		Error     *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResponse.Success {
		errorMsg := "unknown error"
		if apiResponse.Error != nil {
			errorMsg = apiResponse.Error.Message
		}
		return nil, fmt.Errorf("indexing API error: %s", errorMsg)
	}

	if apiResponse.Data == nil {
		return nil, fmt.Errorf("no data in successful response")
	}

	return apiResponse.Data, nil
}

// CreateIndexingQueueItem creates a queue item for indexing
func CreateIndexingQueueItem(
	documentID, documentPath, text string,
	classificationResult *classifier.ClassificationResult,
	options map[string]interface{},
) *queue.QueueItem {
	
	// Extract optional metadata from options
	fileName, _ := options["file_name"].(string)
	contentType, _ := options["content_type"].(string)
	fileURL, _ := options["file_url"].(string)
	sourceJobID, _ := options["source_job_id"].(string)
	size, _ := options["size"].(int64)

	indexingItem := &IndexingQueueItem{
		DocumentID:           documentID,
		DocumentPath:         documentPath,
		Text:                 text,
		ClassificationResult: classificationResult,
		FileName:             fileName,
		ContentType:          contentType,
		Size:                 size,
		FileURL:              fileURL,
		SourceJobID:          sourceJobID,
	}

	return &queue.QueueItem{
		ID:         fmt.Sprintf("index_%s_%d", documentID, time.Now().UnixNano()),
		Type:       queue.QueueTypeIndexing,
		Priority:   queue.PriorityNormal,
		Data:       indexingItem,
		CreatedAt:  time.Now(),
		MaxRetries: 3,
		Metadata: map[string]interface{}{
			"document_id":   documentID,
			"document_path": documentPath,
			"source_job_id": sourceJobID,
		},
	}
}

// GetDefaultIndexingProcessorConfig returns default configuration for the indexing processor
func GetDefaultIndexingProcessorConfig() *IndexingProcessorConfig {
	return &IndexingProcessorConfig{
		APIBaseURL:     "http://localhost:6000",
		RequestTimeout: 30 * time.Second,
		MaxRetries:     3,
		RetryDelay:     5 * time.Second,
	}
}