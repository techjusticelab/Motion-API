# Error Codes Reference

Comprehensive reference for all error codes returned by the Motion Index API.

## Error Response Format

All errors follow a consistent JSON structure:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "additional": "context-specific information"
    },
    "field": "specific_field_name"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## HTTP Status Codes

| Status | Meaning | When Used |
|--------|---------|-----------|
| 400 | Bad Request | Invalid request data, validation failures |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Valid auth but insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists or state conflict |
| 413 | Payload Too Large | File size exceeds limits |
| 415 | Unsupported Media Type | Invalid file format |
| 422 | Unprocessable Entity | Valid format but semantic errors |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Unexpected server error |
| 502 | Bad Gateway | External service unavailable |
| 503 | Service Unavailable | Temporary service maintenance |

## Authentication Errors (401)

### AUTHENTICATION_ERROR
General authentication failure.

```json
{
  "code": "AUTHENTICATION_ERROR",
  "message": "Authentication failed",
  "details": {
    "reason": "invalid_token",
    "token_type": "JWT",
    "expires_at": "2024-01-01T12:00:00Z"
  }
}
```

**Common causes:**
- Missing Authorization header
- Invalid token format
- Expired token
- Malformed JWT

**Resolution:**
- Include valid `Authorization: Bearer <token>` header
- Refresh expired tokens
- Re-authenticate if token is invalid

### MISSING_AUTHORIZATION
No authorization header provided.

```json
{
  "code": "MISSING_AUTHORIZATION",
  "message": "Authorization header is required",
  "details": {
    "required_format": "Bearer <token>",
    "header_name": "Authorization"
  }
}
```

### INVALID_TOKEN_FORMAT
Authorization header format is incorrect.

```json
{
  "code": "INVALID_TOKEN_FORMAT", 
  "message": "Invalid authorization header format",
  "details": {
    "provided": "Token abc123",
    "expected": "Bearer <token>"
  }
}
```

### TOKEN_EXPIRED
JWT token has expired.

```json
{
  "code": "TOKEN_EXPIRED",
  "message": "JWT token has expired",
  "details": {
    "expired_at": "2024-01-01T12:00:00Z",
    "current_time": "2024-01-01T12:30:00Z",
    "suggestion": "Refresh token or re-authenticate"
  }
}
```

## Authorization Errors (403)

### AUTHORIZATION_ERROR
Valid authentication but insufficient permissions.

```json
{
  "code": "AUTHORIZATION_ERROR",
  "message": "Insufficient permissions",
  "details": {
    "required_role": "admin",
    "user_role": "user",
    "action": "delete_document",
    "resource_id": "doc_123"
  }
}
```

### FORBIDDEN_RESOURCE
Access to specific resource is forbidden.

```json
{
  "code": "FORBIDDEN_RESOURCE",
  "message": "Access to this resource is forbidden",
  "details": {
    "resource_type": "document",
    "resource_id": "doc_123",
    "owner_id": "user_456",
    "requester_id": "user_789"
  }
}
```

## Validation Errors (400)

### VALIDATION_ERROR
Request data failed validation.

```json
{
  "code": "VALIDATION_ERROR",
  "message": "Request validation failed",
  "details": {
    "field": "document_type",
    "value": "invalid_type",
    "allowed_values": ["motion", "brief", "order", "transcript"],
    "validation_rule": "enum"
  },
  "field": "document_type"
}
```

### MISSING_REQUIRED_FIELD
Required field is missing from request.

```json
{
  "code": "MISSING_REQUIRED_FIELD",
  "message": "Required field is missing",
  "details": {
    "field": "file",
    "field_type": "multipart/form-data",
    "required_for": "document upload"
  },
  "field": "file"
}
```

### INVALID_FIELD_TYPE
Field has wrong data type.

```json
{
  "code": "INVALID_FIELD_TYPE",
  "message": "Invalid field type",
  "details": {
    "field": "page_count",
    "expected_type": "integer",
    "actual_type": "string",
    "provided_value": "not_a_number"
  },
  "field": "page_count"
}
```

### INVALID_FIELD_VALUE
Field value is invalid.

```json
{
  "code": "INVALID_FIELD_VALUE",
  "message": "Invalid field value",
  "details": {
    "field": "limit",
    "value": 1000,
    "min_value": 1,
    "max_value": 100,
    "constraint": "pagination_limit"
  },
  "field": "limit"
}
```

### INVALID_EMAIL_FORMAT
Email address format is invalid.

```json
{
  "code": "INVALID_EMAIL_FORMAT",
  "message": "Invalid email address format",
  "details": {
    "field": "email",
    "provided_value": "invalid-email",
    "expected_format": "user@domain.com"
  },
  "field": "email"
}
```

### INVALID_DATE_FORMAT
Date format is invalid.

```json
{
  "code": "INVALID_DATE_FORMAT",
  "message": "Invalid date format",
  "details": {
    "field": "filing_date",
    "provided_value": "01/01/2024",
    "expected_format": "YYYY-MM-DD",
    "example": "2024-01-01"
  },
  "field": "filing_date"
}
```

## File Upload Errors (400/413/415)

### FILE_TOO_LARGE
Uploaded file exceeds size limit.

```json
{
  "code": "FILE_TOO_LARGE",
  "message": "File size exceeds maximum limit",
  "details": {
    "filename": "large_document.pdf",
    "file_size": 157286400,
    "max_size": 104857600,
    "max_size_human": "100MB",
    "file_size_human": "150MB"
  }
}
```

### FILE_TOO_SMALL
Uploaded file is too small (likely empty).

```json
{
  "code": "FILE_TOO_SMALL",
  "message": "File is too small or empty",
  "details": {
    "filename": "empty.pdf",
    "file_size": 0,
    "min_size": 1,
    "min_size_human": "1 byte"
  }
}
```

### UNSUPPORTED_FILE_TYPE
File type is not supported.

```json
{
  "code": "UNSUPPORTED_FILE_TYPE",
  "message": "Unsupported file type",
  "details": {
    "filename": "document.xyz",
    "detected_mime_type": "application/octet-stream",
    "file_extension": "xyz",
    "supported_types": ["application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"],
    "supported_extensions": ["pdf", "docx", "txt", "rtf"]
  }
}
```

### CORRUPTED_FILE
File is corrupted or unreadable.

```json
{
  "code": "CORRUPTED_FILE",
  "message": "File appears to be corrupted",
  "details": {
    "filename": "corrupted.pdf",
    "file_size": 1024,
    "issue": "Unable to parse PDF structure",
    "suggestions": ["Re-save the file", "Convert to different format", "Contact support"]
  }
}
```

### VIRUS_DETECTED
File failed virus scan.

```json
{
  "code": "VIRUS_DETECTED",
  "message": "File failed security scan",
  "details": {
    "filename": "malicious.pdf",
    "scanner": "antivirus_engine",
    "threat_type": "malware",
    "action_taken": "file_quarantined"
  }
}
```

## Processing Errors (422/500)

### PROCESSING_ERROR
Document processing failed.

```json
{
  "code": "PROCESSING_ERROR",
  "message": "Document processing failed",
  "details": {
    "document_id": "doc_123",
    "filename": "document.pdf",
    "stage": "text_extraction",
    "reason": "PDF parsing error",
    "technical_details": "Invalid cross-reference table",
    "retry_possible": true,
    "suggestions": ["Try re-uploading", "Use different PDF version", "Contact support"]
  }
}
```

### TEXT_EXTRACTION_FAILED
Text extraction from document failed.

```json
{
  "code": "TEXT_EXTRACTION_FAILED",
  "message": "Failed to extract text from document",
  "details": {
    "document_id": "doc_123",
    "filename": "scanned.pdf",
    "format": "PDF",
    "pages": 10,
    "issue": "Document appears to be scanned images",
    "suggestion": "Enable OCR processing",
    "ocr_available": true
  }
}
```

### CLASSIFICATION_FAILED
AI document classification failed.

```json
{
  "code": "CLASSIFICATION_FAILED",
  "message": "Document classification failed",
  "details": {
    "document_id": "doc_123",
    "filename": "document.pdf",
    "classifier": "gpt-4",
    "reason": "Insufficient text content",
    "extracted_text_length": 50,
    "minimum_required": 100,
    "fallback_category": "unknown"
  }
}
```

### INDEXING_FAILED
Search indexing failed.

```json
{
  "code": "INDEXING_FAILED",
  "message": "Failed to index document for search",
  "details": {
    "document_id": "doc_123",
    "search_engine": "opensearch",
    "reason": "Connection timeout",
    "retry_scheduled": true,
    "retry_at": "2024-01-01T12:05:00Z"
  }
}
```

### STORAGE_ERROR
File storage operation failed.

```json
{
  "code": "STORAGE_ERROR",
  "message": "File storage operation failed",
  "details": {
    "document_id": "doc_123",
    "operation": "upload",
    "storage_backend": "digitalocean_spaces",
    "reason": "Network timeout",
    "retry_possible": true,
    "retry_count": 2,
    "max_retries": 3
  }
}
```

## Search Errors (400/422)

### SEARCH_ERROR
Search operation failed.

```json
{
  "code": "SEARCH_ERROR",
  "message": "Search operation failed",
  "details": {
    "query": "motion to dismiss",
    "search_engine": "opensearch",
    "reason": "Query syntax error",
    "technical_details": "Invalid boolean operator",
    "suggestion": "Check query syntax"
  }
}
```

### INVALID_SEARCH_QUERY
Search query is invalid.

```json
{
  "code": "INVALID_SEARCH_QUERY",
  "message": "Invalid search query",
  "details": {
    "query": "",
    "issue": "Empty query string",
    "min_length": 1,
    "max_length": 500,
    "allowed_characters": "alphanumeric, spaces, quotes, operators"
  }
}
```

### SEARCH_TIMEOUT
Search operation timed out.

```json
{
  "code": "SEARCH_TIMEOUT",
  "message": "Search operation timed out",
  "details": {
    "query": "complex search query",
    "timeout_seconds": 30,
    "suggestion": "Simplify query or add filters to narrow results"
  }
}
```

## Resource Errors (404/409)

### NOT_FOUND
Requested resource was not found.

```json
{
  "code": "NOT_FOUND",
  "message": "Resource not found",
  "details": {
    "resource_type": "document",
    "resource_id": "doc_nonexistent",
    "searched_in": ["database", "storage"],
    "suggestion": "Verify the document ID is correct"
  }
}
```

### DOCUMENT_NOT_FOUND
Specific document was not found.

```json
{
  "code": "DOCUMENT_NOT_FOUND",
  "message": "Document not found",
  "details": {
    "document_id": "doc_123",
    "checked_locations": ["search_index", "file_storage"],
    "possible_reasons": ["Document was deleted", "ID is incorrect", "Access denied"]
  }
}
```

### RESOURCE_CONFLICT
Resource already exists or state conflict.

```json
{
  "code": "RESOURCE_CONFLICT",
  "message": "Resource conflict",
  "details": {
    "resource_type": "document",
    "conflict_type": "duplicate_filename",
    "existing_resource_id": "doc_456",
    "filename": "motion_to_dismiss.pdf",
    "suggestion": "Use different filename or update existing document"
  }
}
```

## Rate Limiting Errors (429)

### RATE_LIMIT_EXCEEDED
Too many requests from client.

```json
{
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Rate limit exceeded",
  "details": {
    "limit": 100,
    "window_seconds": 3600,
    "requests_made": 101,
    "reset_at": "2024-01-01T13:00:00Z",
    "retry_after_seconds": 1800
  }
}
```

### UPLOAD_RATE_LIMIT_EXCEEDED
Too many file uploads.

```json
{
  "code": "UPLOAD_RATE_LIMIT_EXCEEDED",
  "message": "Upload rate limit exceeded",
  "details": {
    "upload_limit": 10,
    "window_minutes": 60,
    "uploads_made": 11,
    "reset_at": "2024-01-01T13:00:00Z",
    "suggestion": "Wait before uploading more files"
  }
}
```

## Service Errors (500/502/503)

### INTERNAL_SERVER_ERROR
Unexpected server error.

```json
{
  "code": "INTERNAL_SERVER_ERROR",
  "message": "An unexpected error occurred",
  "details": {
    "error_id": "err_abc123",
    "timestamp": "2024-01-01T12:00:00Z",
    "suggestion": "Please try again or contact support if the issue persists"
  }
}
```

### SERVICE_UNAVAILABLE
External service is unavailable.

```json
{
  "code": "SERVICE_UNAVAILABLE",
  "message": "Service temporarily unavailable",
  "details": {
    "service": "opensearch",
    "reason": "Connection refused",
    "estimated_recovery": "2024-01-01T12:05:00Z",
    "suggestion": "Please try again in a few minutes"
  }
}
```

### DATABASE_ERROR
Database operation failed.

```json
{
  "code": "DATABASE_ERROR",
  "message": "Database operation failed",
  "details": {
    "operation": "insert",
    "table": "documents",
    "reason": "Connection lost",
    "retry_possible": true
  }
}
```

### EXTERNAL_API_ERROR
External API call failed.

```json
{
  "code": "EXTERNAL_API_ERROR",
  "message": "External API call failed",
  "details": {
    "api": "openai",
    "endpoint": "/v1/chat/completions",
    "status_code": 503,
    "reason": "Service overloaded",
    "retry_after": 60
  }
}
```

## Configuration Errors (500)

### CONFIGURATION_ERROR
Server configuration issue.

```json
{
  "code": "CONFIGURATION_ERROR",
  "message": "Server configuration error",
  "details": {
    "component": "digitalocean_spaces",
    "issue": "Missing access credentials",
    "suggestion": "Contact system administrator"
  }
}
```

### FEATURE_DISABLED
Requested feature is disabled.

```json
{
  "code": "FEATURE_DISABLED",
  "message": "Feature is currently disabled",
  "details": {
    "feature": "ai_classification",
    "reason": "Maintenance mode",
    "estimated_availability": "2024-01-01T14:00:00Z"
  }
}
```

## Error Handling Best Practices

### Client-Side Error Handling

```javascript
async function handleApiCall() {
  try {
    const response = await fetch('/api/v1/search', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({ query: 'motion' })
    });

    const data = await response.json();
    
    if (!response.ok) {
      const error = data.error;
      
      switch (error.code) {
        case 'AUTHENTICATION_ERROR':
          // Redirect to login
          window.location.href = '/login';
          break;
        case 'RATE_LIMIT_EXCEEDED':
          // Show rate limit message and retry after delay
          const retryAfter = error.details.retry_after_seconds;
          setTimeout(() => handleApiCall(), retryAfter * 1000);
          break;
        case 'VALIDATION_ERROR':
          // Show field-specific error
          showFieldError(error.field, error.message);
          break;
        default:
          // Show generic error message
          showError(error.message);
      }
      return;
    }
    
    // Handle success
    processSearchResults(data.data);
    
  } catch (networkError) {
    // Handle network errors
    showError('Network error. Please check your connection.');
  }
}
```

### Retry Logic

```javascript
async function apiCallWithRetry(url, options, maxRetries = 3) {
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      const response = await fetch(url, options);
      const data = await response.json();
      
      if (response.ok) {
        return data;
      }
      
      // Don't retry on client errors (4xx)
      if (response.status >= 400 && response.status < 500) {
        throw new Error(data.error.message);
      }
      
      // Retry on server errors (5xx)
      if (attempt === maxRetries) {
        throw new Error(data.error.message);
      }
      
      // Exponential backoff
      await new Promise(resolve => 
        setTimeout(resolve, Math.pow(2, attempt) * 1000)
      );
      
    } catch (error) {
      if (attempt === maxRetries) {
        throw error;
      }
    }
  }
}
```

## Error Monitoring

### Logging Recommendations
- Log all 4xx and 5xx errors
- Include request ID for correlation
- Log user ID for authenticated requests
- Include relevant context (document ID, search query, etc.)
- Never log sensitive information (passwords, full JWT tokens)

### Alerting Thresholds
- **Critical**: 5xx error rate > 1% for 5 minutes
- **Warning**: 4xx error rate > 10% for 10 minutes
- **Info**: Rate limit exceeded events
- **Info**: Authentication failure patterns