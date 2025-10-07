# Motion Index Fiber API Documentation

## Base URL
- **Development**: `http://localhost:6000`
- **Production**: `https://your-app.ondigitalocean.app`

## Response Format

All endpoints return responses in the following format:

### Success Response
```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": {
    // Response data here
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "error_code",
    "message": "Human readable error message",
    "details": {
      // Additional error details
    },
    "field": "field_name" // For validation errors
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Health & Status Endpoints

### GET /
Root status and service information.

**Response:**
```json
{
  "service": "motion-index-fiber",
  "version": "1.0.0",
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### GET /health
Basic health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "service": "motion-index-fiber"
}
```

## Document Processing & Management

### POST /api/v1/categorise
Upload and process documents with AI classification.

**Content-Type:** `multipart/form-data`

**Parameters:**
- `file` (required): Document file (PDF, DOCX, TXT)
- `category` (optional): Document category (`motion`, `order`, `contract`, `brief`, `memo`, `other`)
- `description` (optional): Document description (max 500 chars)
- `case_name` (optional): Case name (max 200 chars)
- `case_number` (optional): Case number (max 50 chars)
- `author` (optional): Document author (max 100 chars)
- `judge` (optional): Judge name (max 100 chars)
- `court` (optional): Court name (max 200 chars)
- `legal_tags` (optional): Array of legal tags

**Response:**
```json
{
  "success": true,
  "data": {
    "document_id": "doc_123456",
    "file_name": "motion_to_dismiss.pdf",
    "status": "processed",
    "processing_time_ms": 2500,
    "extraction_result": {
      "text": "Extracted text content...",
      "page_count": 10,
      "language": "en"
    },
    "classification_result": {
      "category": "motion",
      "confidence": 0.95,
      "tags": ["motion to dismiss", "criminal defense"]
    },
    "url": "https://spaces.example.com/documents/doc_123456.pdf",
    "cdn_url": "https://cdn.example.com/documents/doc_123456.pdf",
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

### POST /api/v1/analyze-redactions
Analyze PDF redactions for legal compliance.

**Content-Type:** `multipart/form-data`

**Parameters:**
- `file` (required): PDF file to analyze
- `sensitivity` (optional): Analysis sensitivity (`low`, `medium`, `high`)

**Response:**
```json
{
  "success": true,
  "data": {
    "file_name": "redacted_document.pdf",
    "total_pages": 5,
    "redactions_found": 3,
    "redactions": [
      {
        "page": 1,
        "type": "text_block",
        "confidence": 0.98,
        "area": {
          "x": 100,
          "y": 200,
          "width": 150,
          "height": 20
        }
      }
    ],
    "confidence": 0.95,
    "recommendations": ["Review page 1 for potential sensitive information"],
    "analyzed_at": "2024-01-01T12:00:00Z"
  }
}
```

### POST /api/v1/redact-document
Create redacted version of a document.

**Content-Type:** `application/json`

**Body:**
```json
{
  "document_id": "doc_123456",
  "apply_redactions": true
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "document_id": "doc_123456",
    "redacted_url": "https://spaces.example.com/documents/doc_123456_redacted.pdf",
    "message": "Document redacted successfully"
  }
}
```

### POST /api/v1/update-metadata
Update document metadata.

**Content-Type:** `application/json`

**Body:**
```json
{
  "document_id": "doc_123456",
  "metadata": {
    "custom_field": "value"
  },
  "case_name": "Updated Case Name",
  "case_number": "2024-001",
  "author": "Attorney Name",
  "judge": "Judge Name",
  "court": "Superior Court",
  "legal_tags": ["updated", "motion"],
  "status": "approved"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "document_id": "doc_123456",
    "updated_at": "2024-01-01T12:00:00Z",
    "status": "updated"
  }
}
```

### DELETE /api/v1/documents/:id
Delete a document.

**Parameters:**
- `id` (path): Document ID

**Response:**
```json
{
  "success": true,
  "message": "Document deleted successfully"
}
```

## Search & Discovery

### POST /api/v1/search
Advanced document search with legal filtering.

**Content-Type:** `application/json`

**Body:**
```json
{
  "query": "motion to dismiss",
  "filters": {
    "category": "motion",
    "court": "Superior Court",
    "date_range": {
      "start": "2024-01-01",
      "end": "2024-12-31"
    }
  },
  "size": 20,
  "from": 0,
  "sort_by": "relevance",
  "sort_order": "desc",
  "highlight": true
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "query": "motion to dismiss",
    "total_hits": 150,
    "max_score": 2.5,
    "search_time_ms": 45,
    "page": 1,
    "size": 20,
    "documents": [
      {
        "id": "doc_123456",
        "source": {
          "document_name": "motion_to_dismiss.pdf",
          "case_name": "State v. Defendant",
          "category": "motion",
          "author": "Defense Attorney",
          "created_at": "2024-01-01T12:00:00Z"
        },
        "score": 2.5,
        "highlight": {
          "content": ["<em>motion to dismiss</em> the charges..."]
        }
      }
    ],
    "aggregations": {
      "categories": {
        "buckets": [
          {"key": "motion", "doc_count": 100},
          {"key": "order", "doc_count": 50}
        ]
      }
    }
  }
}
```

### GET /api/v1/legal-tags
Get available legal document types and counts.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "tag": "motion to dismiss",
      "count": 25
    },
    {
      "tag": "discovery",
      "count": 18
    }
  ]
}
```

### GET /api/v1/document-types
Get document type classifications.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "type": "motion",
      "count": 150
    },
    {
      "type": "order",
      "count": 75
    }
  ]
}
```

### GET /api/v1/document-stats
Index statistics and analytics.

**Response:**
```json
{
  "success": true,
  "data": {
    "total_documents": 1000,
    "total_size_bytes": 524288000,
    "average_size_bytes": 524288,
    "document_types": [
      {"type": "pdf", "count": 800},
      {"type": "docx", "count": 200}
    ],
    "categories": [
      {"category": "motion", "count": 400},
      {"category": "order", "count": 300}
    ],
    "courts": [
      {"court": "Superior Court", "count": 500},
      {"court": "District Court", "count": 300}
    ],
    "generated_at": "2024-01-01T12:00:00Z"
  }
}
```

### GET /api/v1/field-options
Get available search field options.

**Response:**
```json
{
  "success": true,
  "data": {
    "categories": ["motion", "order", "contract", "brief", "memo", "other"],
    "courts": ["Superior Court", "District Court", "Appeals Court"],
    "authors": ["Attorney A", "Attorney B"],
    "judges": ["Judge Smith", "Judge Johnson"],
    "legal_tags": ["motion to dismiss", "discovery", "sentencing"]
  }
}
```

### GET /api/v1/metadata-fields
Get available metadata fields with types.

**Response:**
```json
{
  "success": true,
  "data": {
    "fields": [
      {"id": "case_name", "name": "Case Name", "type": "string"},
      {"id": "case_number", "name": "Case Number", "type": "string"},
      {"id": "author", "name": "Author", "type": "string"},
      {"id": "judge", "name": "Judge", "type": "string"},
      {"id": "court", "name": "Court", "type": "string"},
      {"id": "legal_tags", "name": "Legal Tags", "type": "array"},
      {"id": "doc_type", "name": "Document Type", "type": "string"},
      {"id": "category", "name": "Category", "type": "string"},
      {"id": "status", "name": "Status", "type": "string"},
      {"id": "created_at", "name": "Created Date", "type": "date"}
    ]
  },
  "message": "Metadata fields retrieved successfully"
}
```

### GET /api/v1/metadata-fields/:field
Get values for specific metadata fields.

**Parameters:**
- `field` (path): Field name
- `prefix` (query, optional): Filter values by prefix
- `size` (query, optional): Number of values to return (default: 50)

**Response:**
```json
{
  "success": true,
  "data": [
    "Superior Court",
    "District Court",
    "Appeals Court"
  ]
}
```

### GET /api/v1/documents/:id
Get specific document details.

**Parameters:**
- `id` (path): Document ID

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "doc_123456",
    "document_name": "motion_to_dismiss.pdf",
    "case_name": "State v. Defendant",
    "case_number": "2024-001",
    "category": "motion",
    "author": "Defense Attorney",
    "judge": "Judge Smith",
    "court": "Superior Court",
    "legal_tags": ["motion to dismiss", "criminal defense"],
    "status": "approved",
    "file_size": 524288,
    "page_count": 10,
    "language": "en",
    "url": "https://spaces.example.com/documents/doc_123456.pdf",
    "cdn_url": "https://cdn.example.com/documents/doc_123456.pdf",
    "created_at": "2024-01-01T12:00:00Z",
    "updated_at": "2024-01-01T12:00:00Z"
  }
}
```

### GET /api/v1/documents/:id/redactions
Get redaction analysis for a document.

**Parameters:**
- `id` (path): Document ID

**Response:**
```json
{
  "success": false,
  "error": {
    "code": "not_found",
    "message": "No redaction analysis found for this document",
    "details": {
      "document_id": "doc_123456"
    }
  }
}
```

## File Storage & CDN

### GET /api/v1/documents/*
Serve documents with automatic CDN redirects.

**Parameters:**
- `*` (path): Document path (e.g., `documents/data/1385.pdf`)

**Response:**
- **Success**: Binary file content with appropriate headers
- **Redirect**: 302 redirect to CDN URL for cloud storage
- **Error**: 404 if document not found

**Headers:**
- `Content-Type`: Appropriate MIME type (e.g., `application/pdf`)
- `Content-Length`: File size in bytes
- `Cache-Control`: Caching directives

## Storage Management

### GET /api/v1/storage/documents
List documents in storage.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `size` (optional): Page size (default: 50)
- `prefix` (optional): Filter by path prefix

**Response:**
```json
{
  "success": true,
  "data": {
    "documents": [
      {
        "key": "documents/doc_123456.pdf",
        "size": 524288,
        "last_modified": "2024-01-01T12:00:00Z",
        "etag": "d41d8cd98f00b204e9800998ecf8427e"
      }
    ],
    "total_count": 1000,
    "page": 1,
    "size": 50
  }
}
```

### GET /api/v1/storage/documents/count
Get document count statistics.

**Response:**
```json
{
  "success": true,
  "data": {
    "total_count": 1000,
    "total_size_bytes": 524288000,
    "by_type": {
      "pdf": 800,
      "docx": 150,
      "txt": 50
    }
  }
}
```

## Batch Processing

### POST /api/v1/batch/classify
Start batch classification job.

**Content-Type:** `multipart/form-data`

**Parameters:**
- `files` (required): Multiple files to process
- `category` (optional): Default category for all files
- `case_name` (optional): Default case name
- `case_number` (optional): Default case number

**Response:**
```json
{
  "success": true,
  "data": {
    "job_id": "batch_123456",
    "status": "queued",
    "file_count": 10,
    "estimated_completion": "2024-01-01T12:05:00Z"
  }
}
```

### GET /api/v1/batch/:job_id/status
Get batch job status.

**Parameters:**
- `job_id` (path): Batch job ID

**Response:**
```json
{
  "success": true,
  "data": {
    "job_id": "batch_123456",
    "status": "processing",
    "progress": {
      "total_files": 10,
      "processed_files": 7,
      "failed_files": 1,
      "progress_percentage": 70
    },
    "started_at": "2024-01-01T12:00:00Z",
    "estimated_completion": "2024-01-01T12:05:00Z"
  }
}
```

### GET /api/v1/batch/:job_id/results
Get batch job results.

**Parameters:**
- `job_id` (path): Batch job ID

**Response:**
```json
{
  "success": true,
  "data": {
    "job_id": "batch_123456",
    "status": "completed",
    "results": [
      {
        "file_name": "document1.pdf",
        "document_id": "doc_123456",
        "status": "success"
      },
      {
        "file_name": "document2.pdf",
        "status": "failed",
        "error": "File format not supported"
      }
    ],
    "completed_at": "2024-01-01T12:05:00Z"
  }
}
```

### DELETE /api/v1/batch/:job_id
Cancel batch job.

**Parameters:**
- `job_id` (path): Batch job ID

**Response:**
```json
{
  "success": true,
  "message": "Batch job cancelled successfully"
}
```

## Document Indexing

### POST /api/v1/index/document
Index a document for search.

**Content-Type:** `application/json`

**Body:**
```json
{
  "document_id": "doc_123456",
  "force_reindex": false
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "document_id": "doc_123456",
    "index_name": "documents",
    "status": "indexed",
    "indexing_time_ms": 150
  }
}
```

## Error Codes

| Code | Description |
|------|-------------|
| `validation_error` | Request validation failed |
| `authentication_error` | Authentication required or failed |
| `authorization_error` | Insufficient permissions |
| `not_found` | Resource not found |
| `file_too_large` | File exceeds maximum size limit |
| `unsupported_format` | File format not supported |
| `processing_error` | Document processing failed |
| `storage_error` | Storage operation failed |
| `search_error` | Search operation failed |
| `rate_limit_exceeded` | Too many requests |
| `internal_error` | Internal server error |

## Rate Limiting

The API implements rate limiting to ensure fair usage:

- **Document Upload**: 10 requests per minute per IP
- **Search**: 100 requests per minute per IP
- **General API**: 1000 requests per hour per IP

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Request limit
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Reset time (Unix timestamp)

## Authentication

Currently, most endpoints are publicly accessible for development. In production, the following endpoints should be protected with JWT authentication:

- `POST /api/v1/update-metadata`
- `DELETE /api/v1/documents/:id`
- `POST /api/v1/batch/*`
- `POST /api/v1/index/document`

**JWT Header Format:**
```
Authorization: Bearer <jwt-token>
```

## Content Types

### Supported Upload Formats
- `application/pdf` - PDF documents
- `application/vnd.openxmlformats-officedocument.wordprocessingml.document` - DOCX files
- `text/plain` - Text files
- `application/rtf` - RTF files

### Response Content Types
- `application/json` - API responses
- `application/pdf` - PDF document serving
- `application/octet-stream` - Binary file serving