# Single-Threaded Document Classification Script

This script provides a sequential, single-threaded approach to document classification that processes documents from DigitalOcean Spaces storage through the processing API and indexes them directly to OpenSearch.

## Features

- **Sequential Processing**: Documents are processed one at a time for easier debugging and monitoring
- **Document Discovery**: Automatically discovers documents from DigitalOcean Spaces via storage API
- **Duplicate Detection**: Checks if documents are already indexed before processing
- **Complete Pipeline**: Downloads → Classifies → Indexes in a single workflow
- **Progress Tracking**: Real-time progress reporting with detailed statistics
- **Error Handling**: Retry logic with detailed error reporting
- **Configurable**: Environment variable configuration for all settings

## Usage

### Commands

```bash
# Test API connectivity
go run cmd/api-classifier/main.go test-connection

# Classify first N documents (default: 10)
go run cmd/api-classifier/main.go classify-count [N]

# Classify ALL documents in storage (sequential)
go run cmd/api-classifier/main.go classify-all
```

### Examples

```bash
# Test that all API endpoints are working
go run cmd/api-classifier/main.go test-connection

# Process 50 documents
go run cmd/api-classifier/main.go classify-count 50

# Process all documents in storage
go run cmd/api-classifier/main.go classify-all
```

## Configuration

Set environment variables in `.env` or export directly:

```bash
# API Configuration
API_BASE_URL=http://localhost:8003          # API base URL
REQUEST_TIMEOUT=120                         # Request timeout in seconds
RETRY_ATTEMPTS=3                           # Number of retry attempts
PROCESSING_DELAY_MS=100                    # Delay between documents in milliseconds
```

## Processing Workflow

For each document, the script:

1. **Check Existence**: Verifies if document is already indexed in OpenSearch
2. **Download**: Downloads document content from DigitalOcean Spaces
3. **Process**: Sends document to processing API for text extraction and classification
4. **Index**: Directly indexes the processed document to OpenSearch
5. **Progress**: Reports success/failure and updates statistics

## Key Differences from Batch Classifier

| Feature | Batch Classifier | Single-Threaded Classifier |
|---------|-----------------|---------------------------|
| Concurrency | Multi-threaded workers | Single-threaded sequential |
| Processing | Batch job API | Individual processing API |
| Queue | Background job queue | Direct processing |
| Monitoring | Job status polling | Real-time progress |
| Debugging | Complex multi-worker logs | Simple sequential logs |
| Error Handling | Batch failure recovery | Individual document retry |

## Output

The script provides detailed progress information:

```
🚀 Single-Threaded Classification of 50 Documents
===============================================
📊 Retrieved 50 documents for processing

🔄 [1/50] Processing: documents/case_001.pdf
   📥 Downloading document content...
   🤖 Classifying document...
   📦 Indexing to OpenSearch...
✅ [1/50] Successfully processed: documents/case_001.pdf

🔄 [2/50] Processing: documents/case_002.pdf
   ⏭️  Document already indexed, skipping: documents/case_002.pdf
✅ [2/50] Successfully processed: documents/case_002.pdf

📊 SINGLE-THREADED CLASSIFICATION COMPLETE
==========================================
⏱️  Total Processing Time: 2m30s
📁 Total Documents Found: 50
✅ Successfully Processed: 48
❌ Failed Documents: 2
📋 Total Processed: 50
⚡ Average Rate: 20.00 documents/minute
📈 Success Rate: 96.0%
```

## Use Cases

- **Development**: Test classification changes with controlled, sequential processing
- **Debugging**: Isolate issues with individual documents without worker complexity
- **Maintenance**: Process specific document sets with detailed monitoring
- **Recovery**: Re-process failed documents from batch operations
- **Testing**: Validate processing pipeline with small document sets

## Error Handling

- **Network Errors**: Automatic retry with exponential backoff
- **Processing Failures**: Individual document failure doesn't stop the batch
- **Index Errors**: Detailed error logging for troubleshooting
- **Duplicate Handling**: Skips already-indexed documents to avoid duplication

## Performance Considerations

- **Sequential Processing**: Slower than batch processing but easier to debug
- **Memory Efficient**: Processes documents one at a time
- **Network Efficient**: Reuses HTTP connections where possible
- **Rate Limiting**: Configurable delays between documents to avoid API limits