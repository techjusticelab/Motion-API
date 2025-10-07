# Search & Discovery Endpoints

Advanced document search with legal-specific filtering, aggregations, and metadata discovery.

## Search Documents

### `POST /api/v1/search`
Perform full-text search with advanced filtering for legal documents.

**Request Body:**
```json
{
  "query": "motion to dismiss",
  "filters": {
    "document_type": ["motion", "brief"],
    "court": "Superior Court of California",
    "date_range": {
      "start": "2024-01-01",
      "end": "2024-12-31"
    },
    "case_id": "2024-CV-12345"
  },
  "sort": {
    "field": "relevance",
    "order": "desc"
  },
  "page": 1,
  "limit": 20,
  "highlight": true,
  "aggregations": ["document_type", "court", "judge"]
}
```

**Parameters:**
- `query` (string): Full-text search query
- `filters` (object): Filtering criteria
  - `document_type` (array): Filter by document types
  - `court` (string): Filter by court name
  - `judge` (string): Filter by judge name
  - `date_range` (object): Date range filter
  - `case_id` (string): Filter by case ID
- `sort` (object): Sorting criteria
  - `field` (string): Sort field (`relevance`, `date`, `title`)
  - `order` (string): Sort order (`asc`, `desc`)
- `page` (integer): Page number (default: 1)
- `limit` (integer): Results per page (default: 20, max: 100)
- `highlight` (boolean): Include search term highlighting
- `aggregations` (array): Include aggregation data

**Response:**
```json
{
  "success": true,
  "message": "Search completed",
  "data": {
    "total": 247,
    "page": 1,
    "limit": 20,
    "pages": 13,
    "took": "42ms",
    "documents": [
      {
        "id": "doc_12345",
        "title": "Motion to Dismiss - Smith v. Johnson",
        "content_preview": "Defendant hereby moves this Court to dismiss the complaint...",
        "document_type": "motion",
        "court": "Superior Court of California, County of San Francisco",
        "case_id": "2024-CV-12345",
        "judge": "Hon. Jane Smith",
        "filing_date": "2024-03-15T00:00:00Z",
        "created_at": "2024-03-16T10:30:00Z",
        "url": "https://cdn.motionindex.com/documents/doc_12345.pdf",
        "score": 0.95,
        "highlights": [
          "Defendant hereby moves this Court to <em>dismiss</em> the complaint",
          "This <em>motion</em> is based on the following grounds"
        ],
        "metadata": {
          "pages": 12,
          "file_size": 245760,
          "language": "en",
          "parties": ["Smith", "Johnson"],
          "attorneys": ["John Doe, Esq.", "Jane Attorney"]
        }
      }
    ],
    "aggregations": {
      "document_type": {
        "motion": 145,
        "brief": 67,
        "order": 23,
        "transcript": 12
      },
      "court": {
        "Superior Court of California, County of San Francisco": 156,
        "Superior Court of California, County of Los Angeles": 91
      },
      "judge": {
        "Hon. Jane Smith": 89,
        "Hon. John Doe": 76,
        "Hon. Mary Johnson": 54
      }
    }
  }
}
```

## Discovery Endpoints

### `GET /api/v1/legal-tags`
Get available legal document types and their counts.

**Response:**
```json
{
  "success": true,
  "message": "Legal tags retrieved",
  "data": {
    "document_types": [
      {
        "type": "motion",
        "label": "Motion",
        "count": 1453,
        "subcategories": [
          {
            "type": "motion_to_dismiss",
            "label": "Motion to Dismiss",
            "count": 342
          },
          {
            "type": "motion_for_summary_judgment",
            "label": "Motion for Summary Judgment", 
            "count": 178
          }
        ]
      },
      {
        "type": "brief",
        "label": "Brief",
        "count": 897,
        "subcategories": [
          {
            "type": "appellate_brief",
            "label": "Appellate Brief",
            "count": 234
          }
        ]
      }
    ],
    "practice_areas": [
      {
        "area": "criminal_law",
        "label": "Criminal Law",
        "count": 1867
      },
      {
        "area": "civil_litigation",
        "label": "Civil Litigation",
        "count": 1234
      }
    ]
  }
}
```

### `GET /api/v1/document-types`
Get document type classifications and metadata.

**Response:**
```json
{
  "success": true,
  "message": "Document types retrieved",
  "data": {
    "types": [
      {
        "id": "motion",
        "name": "Motion",
        "description": "A formal request to a court for a ruling or order",
        "count": 1453,
        "common_subtypes": ["motion_to_dismiss", "motion_for_summary_judgment"],
        "typical_length": "5-20 pages",
        "filing_requirements": "Must be served on all parties"
      }
    ]
  }
}
```

### `GET /api/v1/document-stats`
Get index statistics and analytics.

**Response:**
```json
{
  "success": true,
  "message": "Document statistics",
  "data": {
    "total_documents": 23547,
    "total_size": "15.7GB",
    "index_health": "green",
    "last_updated": "2024-01-01T12:00:00Z",
    "breakdown": {
      "by_type": {
        "motion": 8523,
        "brief": 6745,
        "order": 4234,
        "transcript": 2987,
        "other": 1058
      },
      "by_year": {
        "2024": 12456,
        "2023": 8765,
        "2022": 2326
      },
      "by_court": {
        "Superior Court of California": 18234,
        "Court of Appeal": 3456,
        "Federal District Court": 1857
      }
    },
    "processing_stats": {
      "avg_processing_time": "2.3s",
      "successful_extractions": 0.987,
      "classification_accuracy": 0.943
    }
  }
}
```

### `GET /api/v1/field-options`
Get available search field options and filters.

**Response:**
```json
{
  "success": true,
  "message": "Field options retrieved",
  "data": {
    "filterable_fields": [
      {
        "field": "document_type",
        "type": "categorical",
        "values": ["motion", "brief", "order", "transcript"],
        "description": "Type of legal document"
      },
      {
        "field": "court",
        "type": "text",
        "searchable": true,
        "description": "Court where document was filed"
      },
      {
        "field": "filing_date",
        "type": "date",
        "range_supported": true,
        "description": "Date document was filed"
      }
    ],
    "sortable_fields": [
      {
        "field": "relevance",
        "default": true,
        "description": "Search relevance score"
      },
      {
        "field": "filing_date",
        "description": "Document filing date"
      },
      {
        "field": "title",
        "description": "Document title alphabetical"
      }
    ]
  }
}
```

### `GET /api/v1/metadata-fields/:field`
Get unique values for a specific metadata field.

**Parameters:**
- `field` (string): Field name (e.g., "judge", "court", "attorney")

**Query Parameters:**
- `search` (string): Filter values by search term
- `limit` (integer): Maximum values to return (default: 100)

**Example:** `GET /api/v1/metadata-fields/judge?search=smith&limit=10`

**Response:**
```json
{
  "success": true,
  "message": "Field values retrieved",
  "data": {
    "field": "judge",
    "total": 156,
    "values": [
      {
        "value": "Hon. Jane Smith",
        "count": 89,
        "last_seen": "2024-01-01T12:00:00Z"
      },
      {
        "value": "Hon. John Smith",
        "count": 34,
        "last_seen": "2023-12-28T14:30:00Z"
      }
    ]
  }
}
```

### `GET /api/v1/documents/:id`
Get specific document details and metadata.

**Parameters:**
- `id` (string): Document ID

**Response:**
```json
{
  "success": true,
  "message": "Document retrieved",
  "data": {
    "id": "doc_12345",
    "title": "Motion to Dismiss - Smith v. Johnson",
    "content": "Full document text content...",
    "document_type": "motion",
    "court": "Superior Court of California, County of San Francisco",
    "case_id": "2024-CV-12345",
    "judge": "Hon. Jane Smith",
    "filing_date": "2024-03-15T00:00:00Z",
    "created_at": "2024-03-16T10:30:00Z",
    "updated_at": "2024-03-16T10:30:00Z",
    "url": "https://cdn.motionindex.com/documents/doc_12345.pdf",
    "download_url": "https://cdn.motionindex.com/documents/doc_12345.pdf?download=true",
    "metadata": {
      "pages": 12,
      "file_size": 245760,
      "format": "PDF",
      "language": "en",
      "parties": ["Smith", "Johnson"],
      "attorneys": ["John Doe, Esq.", "Jane Attorney"],
      "docket_number": "24-CV-12345",
      "classification_confidence": 0.94,
      "extracted_entities": [
        {
          "type": "person",
          "value": "John Smith",
          "confidence": 0.98
        }
      ],
      "redaction_analysis": {
        "redacted_sections": 3,
        "compliance_score": 0.95,
        "issues": []
      }
    },
    "related_documents": [
      {
        "id": "doc_12346",
        "title": "Opposition to Motion to Dismiss",
        "relationship": "response",
        "similarity": 0.87
      }
    ]
  }
}
```

## Error Responses

### Validation Errors
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid search parameters",
    "details": {
      "field": "limit",
      "value": 150,
      "message": "limit must be between 1 and 100"
    }
  }
}
```

### Search Service Errors
```json
{
  "success": false,
  "error": {
    "code": "SEARCH_ERROR",
    "message": "Search service temporarily unavailable",
    "details": {
      "service": "opensearch",
      "status": "degraded",
      "retry_after": 30
    }
  }
}
```

## Search Tips

### Query Syntax
- **Phrase search**: `"motion to dismiss"`
- **Boolean operators**: `motion AND dismiss`
- **Wildcard**: `motion*`
- **Field search**: `title:"motion to dismiss"`
- **Fuzzy search**: `motion~`

### Performance Optimization
- Use specific filters to narrow results
- Limit result size for faster responses
- Use pagination for large result sets
- Cache common search queries

### Best Practices
1. **Use filters**: Always apply relevant filters to improve performance
2. **Pagination**: Don't request more than 50 results at once
3. **Caching**: Cache search results when appropriate
4. **Monitoring**: Track search performance and user patterns