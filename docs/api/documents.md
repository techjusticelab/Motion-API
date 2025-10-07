# Document Processing Endpoints

Upload, process, and analyze legal documents with AI-powered classification and text extraction.

## Document Upload & Classification

### `POST /api/v1/categorise`
Upload and process legal documents with automatic classification and metadata extraction.

**Content-Type:** `multipart/form-data`

**Form Fields:**
- `file` (file): Document file (PDF, DOCX, TXT, RTF)
- `metadata` (string): JSON metadata object (optional)
- `case_id` (string): Associated case ID (optional)
- `document_type` (string): Manual document type override (optional)
- `processing_options` (string): JSON processing options (optional)

**Request Example:**
```bash
curl -X POST http://localhost:6000/api/v1/categorise \
  -F "file=@motion_to_dismiss.pdf" \
  -F "metadata={\"case_id\":\"2024-CV-12345\",\"court\":\"Superior Court of California\"}" \
  -F "document_type=motion" \
  -F "processing_options={\"extract_entities\":true,\"analyze_redactions\":true}"
```

**Metadata Object Schema:**
```json
{
  "case_id": "2024-CV-12345",
  "court": "Superior Court of California, County of San Francisco",
  "judge": "Hon. Jane Smith",
  "docket_number": "24-CV-12345",
  "filing_date": "2024-03-15",
  "parties": ["John Smith", "ABC Corporation"],
  "attorneys": ["Jane Doe, Esq.", "John Attorney"],
  "practice_area": "civil_litigation",
  "custom_fields": {
    "internal_id": "INT-2024-001",
    "priority": "high"
  }
}
```

**Processing Options Schema:**
```json
{
  "extract_entities": true,
  "analyze_redactions": true,
  "generate_summary": false,
  "ocr_enabled": true,
  "classification_confidence_threshold": 0.8,
  "language": "en"
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "Document processed successfully",
  "data": {
    "document_id": "doc_12345",
    "title": "Motion to Dismiss - Smith v. Johnson",
    "document_type": "motion",
    "classification": {
      "primary_type": "motion",
      "subtype": "motion_to_dismiss",
      "confidence": 0.94,
      "alternatives": [
        {
          "type": "brief",
          "confidence": 0.23
        }
      ]
    },
    "processing_results": {
      "text_extraction": {
        "status": "success",
        "extracted_text_length": 15420,
        "language": "en",
        "processing_time": "1.2s"
      },
      "ai_classification": {
        "status": "success",
        "model": "gpt-4",
        "processing_time": "0.8s"
      },
      "entity_extraction": {
        "status": "success",
        "entities": [
          {
            "type": "person",
            "value": "John Smith",
            "confidence": 0.98,
            "positions": [{"start": 125, "end": 135}]
          },
          {
            "type": "organization",
            "value": "ABC Corporation",
            "confidence": 0.95,
            "positions": [{"start": 200, "end": 215}]
          }
        ]
      },
      "redaction_analysis": {
        "status": "success",
        "redacted_sections": 0,
        "compliance_score": 1.0,
        "issues": []
      }
    },
    "file_info": {
      "original_filename": "motion_to_dismiss.pdf",
      "file_size": 245760,
      "format": "PDF",
      "pages": 12,
      "checksum": "sha256:abc123..."
    },
    "storage": {
      "url": "https://cdn.motionindex.com/documents/doc_12345.pdf",
      "cdn_enabled": true,
      "expires_at": null
    },
    "indexing": {
      "status": "success",
      "indexed_at": "2024-01-01T12:01:30Z",
      "searchable": true
    },
    "created_at": "2024-01-01T12:00:00Z",
    "processing_time": "2.3s"
  }
}
```

**Response (Processing Error):**
```json
{
  "success": false,
  "error": {
    "code": "PROCESSING_ERROR",
    "message": "Document processing failed",
    "details": {
      "stage": "text_extraction",
      "reason": "Corrupted PDF file",
      "file_info": {
        "filename": "document.pdf",
        "size": 245760
      },
      "suggestions": [
        "Try re-saving the PDF",
        "Convert to a different format",
        "Contact support if problem persists"
      ]
    }
  }
}
```

## PDF Redaction Analysis

### `POST /api/v1/analyze-redactions`
Analyze PDF documents for redaction compliance and potential issues.

**Content-Type:** `multipart/form-data`

**Form Fields:**
- `file` (file): PDF document to analyze
- `compliance_standard` (string): Legal compliance standard (optional, default: "california_public_defender")
- `detailed_analysis` (boolean): Include detailed analysis report (optional, default: false)

**Request Example:**
```bash
curl -X POST http://localhost:6000/api/v1/analyze-redactions \
  -F "file=@redacted_document.pdf" \
  -F "compliance_standard=california_public_defender" \
  -F "detailed_analysis=true"
```

**Response (Compliant Document):**
```json
{
  "success": true,
  "message": "Redaction analysis completed",
  "data": {
    "document_id": "doc_12346",
    "analysis_id": "analysis_789",
    "compliance": {
      "overall_score": 0.95,
      "status": "compliant",
      "standard": "california_public_defender",
      "last_updated": "2024-01-01T12:00:00Z"
    },
    "redactions": {
      "total_count": 15,
      "by_type": {
        "personal_information": 8,
        "financial_data": 3,
        "sensitive_details": 4
      },
      "by_page": {
        "1": 3,
        "2": 7,
        "3": 5
      }
    },
    "findings": {
      "issues": [],
      "warnings": [
        {
          "type": "partial_redaction",
          "severity": "low",
          "page": 2,
          "description": "Partial name visible at end of redaction block",
          "recommendation": "Extend redaction by 2 characters"
        }
      ],
      "compliant_practices": [
        {
          "type": "complete_ssn_redaction",
          "description": "All social security numbers properly redacted"
        }
      ]
    },
    "detailed_analysis": {
      "redaction_quality": {
        "avg_coverage": 0.98,
        "consistent_styling": true,
        "proper_blocking": true
      },
      "potential_leaks": [],
      "metadata_check": {
        "title_clean": true,
        "author_redacted": true,
        "creation_date_present": true,
        "custom_properties_clean": true
      }
    },
    "recommendations": [
      "Consider extending redaction on page 2",
      "Review metadata for any sensitive information"
    ],
    "processing_time": "3.1s"
  }
}
```

**Response (Non-Compliant Document):**
```json
{
  "success": true,
  "message": "Redaction analysis completed",
  "data": {
    "compliance": {
      "overall_score": 0.65,
      "status": "non_compliant",
      "standard": "california_public_defender"
    },
    "findings": {
      "issues": [
        {
          "type": "exposed_ssn",
          "severity": "critical",
          "page": 1,
          "position": {"x": 150, "y": 300},
          "description": "Social Security Number visible: ***-**-1234",
          "recommendation": "Apply complete redaction to SSN"
        },
        {
          "type": "partial_name_exposure", 
          "severity": "high",
          "page": 3,
          "description": "Last name partially visible under redaction",
          "recommendation": "Extend redaction block"
        }
      ],
      "warnings": [
        {
          "type": "metadata_exposure",
          "severity": "medium",
          "description": "Document contains author name in metadata",
          "recommendation": "Clean document metadata"
        }
      ]
    },
    "required_actions": [
      "Redact exposed SSN on page 1",
      "Extend name redaction on page 3", 
      "Clean document metadata"
    ]
  }
}
```

## File Upload Limits & Formats

### Supported Formats
| Format | MIME Type | Max Size | Features |
|--------|-----------|----------|----------|
| PDF | `application/pdf` | 100MB | Full text extraction, redaction analysis |
| DOCX | `application/vnd.openxmlformats-officedocument.wordprocessingml.document` | 50MB | Full text extraction, metadata |
| TXT | `text/plain` | 10MB | Direct text processing |
| RTF | `application/rtf` | 25MB | Formatted text extraction |

### Processing Features by Format

#### PDF Documents
- ✅ Full text extraction (including OCR for scanned documents)
- ✅ Redaction analysis and compliance checking
- ✅ Metadata extraction (author, creation date, etc.)
- ✅ Page count and structure analysis
- ✅ Entity extraction from text content

#### DOCX Documents  
- ✅ Full text extraction with formatting preservation
- ✅ Metadata extraction (author, company, etc.)
- ✅ Comment and revision history extraction
- ✅ Entity extraction from text content
- ❌ Redaction analysis (not applicable)

#### Plain Text (TXT)
- ✅ Direct text processing
- ✅ Entity extraction
- ✅ Language detection
- ❌ Metadata extraction (limited file info only)
- ❌ Redaction analysis (not applicable)

#### Rich Text Format (RTF)
- ✅ Text extraction with basic formatting
- ✅ Embedded object detection
- ✅ Entity extraction
- ❌ Advanced metadata extraction
- ❌ Redaction analysis (not applicable)

## Error Handling

### File Upload Errors
```json
{
  "success": false,
  "error": {
    "code": "UPLOAD_ERROR",
    "message": "File upload failed",
    "details": {
      "reason": "file_too_large",
      "max_size": "100MB",
      "actual_size": "150MB",
      "filename": "large_document.pdf"
    }
  }
}
```

### Processing Errors
```json
{
  "success": false,
  "error": {
    "code": "PROCESSING_ERROR", 
    "message": "Document processing failed",
    "details": {
      "stage": "ai_classification",
      "reason": "service_unavailable",
      "retry_after": 60,
      "document_id": "doc_12345"
    }
  }
}
```

### Format Errors
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Unsupported file format",
    "details": {
      "filename": "document.xyz",
      "detected_type": "application/octet-stream",
      "supported_formats": ["application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "text/plain", "application/rtf"]
    }
  }
}
```

## Processing Pipeline

### Stage 1: File Upload & Validation
1. File format validation
2. Size limit checking
3. Virus scanning (if enabled)
4. Checksum calculation
5. Storage to DigitalOcean Spaces

### Stage 2: Text Extraction
1. Format-specific extraction (PDF, DOCX, etc.)
2. OCR for scanned documents (PDFs)
3. Language detection
4. Text cleaning and normalization

### Stage 3: AI Classification
1. OpenAI GPT-4 analysis
2. Document type classification
3. Confidence scoring
4. Alternative classification suggestions

### Stage 4: Entity Extraction
1. Named entity recognition
2. Legal entity identification (parties, attorneys, courts)
3. Date and case number extraction
4. Custom field extraction

### Stage 5: Specialized Analysis
1. Redaction analysis (PDFs only)
2. Compliance checking
3. Metadata extraction and cleaning
4. Security scanning

### Stage 6: Indexing & Storage
1. OpenSearch indexing
2. CDN distribution
3. Metadata storage
4. Search optimization

## Performance Metrics

### Processing Times (Typical)
- **Small documents** (<1MB): 1-3 seconds
- **Medium documents** (1-10MB): 3-8 seconds  
- **Large documents** (10-50MB): 8-20 seconds
- **Very large documents** (50-100MB): 20-60 seconds

### Accuracy Rates
- **Text extraction**: 99.5% for digital PDFs, 95% for scanned PDFs
- **Document classification**: 94% accuracy with 0.8+ confidence
- **Entity extraction**: 92% precision, 88% recall
- **Redaction detection**: 97% accuracy for standard redaction patterns

### Throughput
- **Concurrent uploads**: Up to 10 simultaneous per user
- **System capacity**: 100 documents/minute peak
- **Queue processing**: 30-second average wait time during peak