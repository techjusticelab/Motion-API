# Motion Index API Documentation

Motion Index Fiber API provides high-performance legal document processing, search, and storage capabilities designed for California public defenders.

## API Overview

- **Base URL**: `https://your-app.ondigitalocean.app` (production) or `http://localhost:6000` (development)
- **API Version**: v1
- **Content-Type**: `application/json` (unless specified otherwise)
- **Authentication**: JWT Bearer tokens for protected endpoints

## Quick Start

### Health Check
```bash
curl http://localhost:6000/health
```

### Search Documents
```bash
curl -X POST http://localhost:6000/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "motion to dismiss",
    "filters": {
      "document_type": "motion"
    },
    "limit": 10
  }'
```

### Upload Document
```bash
curl -X POST http://localhost:6000/api/v1/categorise \
  -F "file=@document.pdf" \
  -F "metadata={\"case_id\":\"12345\",\"document_type\":\"motion\"}"
```

## Endpoint Categories

### 🔍 [Health & Status](./health.md)
Monitor system health and performance metrics
- `GET /` - Root status
- `GET /health` - Basic health check  
- `GET /health/detailed` - Comprehensive status
- `GET /health/ready` - Readiness probe
- `GET /health/live` - Liveness probe
- `GET /metrics` - Application metrics

### 📄 [Document Processing](./documents.md)
Upload, process, and analyze legal documents
- `POST /api/v1/categorise` - Upload and classify documents
- `POST /api/v1/analyze-redactions` - Analyze PDF redactions

### 🔍 [Search & Discovery](./search.md)
Advanced search with legal-specific filtering
- `POST /api/v1/search` - Document search
- `GET /api/v1/legal-tags` - Available legal types
- `GET /api/v1/document-types` - Document classifications
- `GET /api/v1/document-stats` - Index statistics
- `GET /api/v1/field-options` - Search field options
- `GET /api/v1/metadata-fields/:field` - Field values
- `GET /api/v1/documents/:id` - Document details

### 📁 [File Storage](./storage.md)
CDN-optimized file serving and management
- `GET /api/v1/documents/*` - Serve documents

### 🔒 [Protected Endpoints](./auth.md)
Authentication-required operations
- `POST /api/v1/update-metadata` - Update document metadata
- `DELETE /api/v1/documents/:id` - Delete documents

## Authentication

### JWT Bearer Tokens
Protected endpoints require a valid JWT token in the Authorization header:

```bash
curl -X POST http://localhost:6000/api/v1/update-metadata \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{"document_id": "12345", "metadata": {...}}'
```

### Token Format
```
Authorization: Bearer <jwt-token>
```

JWT tokens are issued by Supabase and contain user identity and permissions.

## Request/Response Format

### Standard Success Response
```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": {
    // Response data
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Standard Error Response
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      // Additional error context
    }
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Common HTTP Status Codes
- `200` - Success
- `201` - Created
- `400` - Bad Request (validation error)
- `401` - Unauthorized (missing or invalid token)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found
- `422` - Unprocessable Entity (semantic error)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error
- `503` - Service Unavailable (maintenance or overload)

## Rate Limiting

- **Default Limit**: 1000 requests per hour per IP
- **Document Upload**: 100 uploads per hour per IP
- **Search Queries**: 500 searches per hour per IP

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

## Error Handling

### Common Error Codes
- `VALIDATION_ERROR` - Input validation failed
- `AUTHENTICATION_ERROR` - Invalid or missing authentication
- `AUTHORIZATION_ERROR` - Insufficient permissions
- `NOT_FOUND` - Requested resource not found
- `SERVICE_UNAVAILABLE` - External service temporarily unavailable
- `PROCESSING_ERROR` - Document processing failed
- `STORAGE_ERROR` - File storage operation failed
- `SEARCH_ERROR` - Search operation failed

### Error Response Examples

#### Validation Error
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request data",
    "details": {
      "field": "document_type",
      "value": "invalid_type",
      "allowed_values": ["motion", "brief", "order", "transcript"]
    }
  }
}
```

#### Authentication Error
```json
{
  "success": false,
  "error": {
    "code": "AUTHENTICATION_ERROR",
    "message": "Invalid or expired token",
    "details": {
      "token_expired": true,
      "expires_at": "2024-01-01T12:00:00Z"
    }
  }
}
```

## Content Types

### Supported Upload Formats
- **PDF**: `application/pdf`
- **DOCX**: `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
- **TXT**: `text/plain`
- **RTF**: `application/rtf`

### File Size Limits
- **Maximum File Size**: 100MB per file
- **Batch Upload**: Up to 10 files per request
- **Total Batch Size**: 500MB maximum

## Performance Guidelines

### Pagination
For endpoints returning lists, use pagination parameters:
```json
{
  "page": 1,
  "limit": 20,
  "total": 1500,
  "pages": 75
}
```

### Caching
- **Document URLs**: Cached for 24 hours
- **Search Results**: Cached for 5 minutes
- **Metadata**: Cached for 1 hour

### Best Practices
1. **Use appropriate page sizes**: Default 20, maximum 100
2. **Cache responses locally** when appropriate
3. **Use ETags** for conditional requests
4. **Batch operations** when possible
5. **Monitor rate limits** to avoid throttling

## SDK and Libraries

### Official SDKs
- **JavaScript/TypeScript**: Coming soon
- **Python**: Coming soon
- **Go**: Native (this repository)

### Community Libraries
Contributions welcome! Please see [Contributing Guidelines](../../CONTRIBUTING.md).

## OpenAPI Specification

The complete OpenAPI 3.0 specification is available at:
- **Development**: `http://localhost:6000/api/docs`
- **Production**: `https://your-app.ondigitalocean.app/api/docs`

Download the spec: [openapi.yaml](./openapi.yaml)

## Code Examples

### JavaScript/TypeScript
```javascript
// Search for documents
const searchResponse = await fetch('/api/v1/search', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    query: 'motion to dismiss',
    filters: {
      document_type: ['motion'],
      date_range: {
        start: '2024-01-01',
        end: '2024-12-31'
      }
    },
    limit: 20
  })
});

const searchData = await searchResponse.json();
console.log(`Found ${searchData.data.total} documents`);

// Upload document with authentication
const formData = new FormData();
formData.append('file', fileInput.files[0]);
formData.append('metadata', JSON.stringify({
  case_id: '2024-CV-001',
  court: 'Superior Court of California'
}));

const uploadResponse = await fetch('/api/v1/categorise', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${jwtToken}`
  },
  body: formData
});
```

### Python
```python
import requests
import json

# Search documents
search_payload = {
    'query': 'habeas corpus',
    'filters': {
        'document_type': ['brief', 'motion'],
        'court': 'Superior Court of California'
    },
    'limit': 50
}

response = requests.post(
    'http://localhost:6000/api/v1/search',
    json=search_payload
)

if response.status_code == 200:
    results = response.json()
    print(f"Found {results['data']['total']} documents")
    for doc in results['data']['documents']:
        print(f"- {doc['title']} ({doc['document_type']})")

# Upload document with authentication
files = {'file': open('document.pdf', 'rb')}
data = {
    'metadata': json.dumps({
        'case_id': '2024-CV-002',
        'document_type': 'motion'
    })
}
headers = {'Authorization': f'Bearer {jwt_token}'}

upload_response = requests.post(
    'http://localhost:6000/api/v1/categorise',
    files=files,
    data=data,
    headers=headers
)

print(f"Upload status: {upload_response.status_code}")
```

### cURL Examples
```bash
# Health check
curl -s http://localhost:6000/health | jq '.'

# Search with complex filters
curl -X POST http://localhost:6000/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "motion to suppress evidence",
    "filters": {
      "document_type": ["motion"],
      "court": "Superior Court of California",
      "date_range": {
        "start": "2023-01-01",
        "end": "2024-01-01"
      }
    },
    "sort": {
      "field": "filing_date", 
      "order": "desc"
    },
    "limit": 10,
    "include_aggregations": true
  }' | jq '.'

# Upload document with authentication
curl -X POST http://localhost:6000/api/v1/categorise \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -F "file=@motion.pdf" \
  -F "metadata={\"case_id\":\"2024-CV-001\",\"court\":\"Superior Court\"}" \
  -F "processing_options={\"extract_entities\":true}" | jq '.'

# Update document metadata (requires auth)
curl -X POST http://localhost:6000/api/v1/update-metadata \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "document_id": "doc_12345",
    "metadata": {
      "judge": "Hon. Jane Smith",
      "case_number": "CV-2024-001234",
      "tags": ["motion", "civil", "pretrial"]
    }
  }' | jq '.'

# Analyze PDF redactions
curl -X POST http://localhost:6000/api/v1/analyze-redactions \
  -F "file=@redacted_document.pdf" \
  -F "detailed_analysis=true" | jq '.data.compliance'
```

### Go Example (Client Library)
```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
)

type SearchRequest struct {
    Query   string                 `json:"query"`
    Filters map[string]interface{} `json:"filters"`
    Limit   int                    `json:"limit"`
}

type SearchResponse struct {
    Success bool `json:"success"`
    Data    struct {
        Total     int `json:"total"`
        Documents []struct {
            ID    string `json:"id"`
            Title string `json:"title"`
            Type  string `json:"document_type"`
        } `json:"documents"`
    } `json:"data"`
}

func main() {
    // Search example
    search := SearchRequest{
        Query: "motion to dismiss",
        Filters: map[string]interface{}{
            "document_type": []string{"motion"},
        },
        Limit: 20,
    }

    searchData, _ := json.Marshal(search)
    resp, err := http.Post(
        "http://localhost:6000/api/v1/search",
        "application/json",
        bytes.NewBuffer(searchData),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    var result SearchResponse
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Printf("Found %d documents\n", result.Data.Total)

    // Upload example
    file, _ := os.Open("document.pdf")
    defer file.Close()

    var buf bytes.Buffer
    writer := multipart.NewWriter(&buf)
    
    part, _ := writer.CreateFormFile("file", "document.pdf")
    io.Copy(part, file)
    
    writer.WriteField("metadata", `{"case_id":"2024-CV-001"}`)
    writer.Close()

    req, _ := http.NewRequest("POST", "http://localhost:6000/api/v1/categorise", &buf)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    req.Header.Set("Authorization", "Bearer "+jwtToken)

    client := &http.Client{}
    uploadResp, _ := client.Do(req)
    defer uploadResp.Body.Close()

    fmt.Printf("Upload status: %d\n", uploadResp.StatusCode)
}
```

## Integration Patterns

### Webhook Integration
```javascript
// Set up webhook endpoint to receive processing notifications
app.post('/webhook/document-processed', (req, res) => {
  const { document_id, status, processing_results } = req.body;
  
  if (status === 'success') {
    console.log(`Document ${document_id} processed successfully`);
    // Update your application state
    updateDocumentStatus(document_id, processing_results);
  } else {
    console.error(`Processing failed for ${document_id}`);
    // Handle processing errors
    handleProcessingError(document_id, processing_results.error);
  }
  
  res.status(200).send('OK');
});
```

### Batch Processing
```python
import asyncio
import aiohttp
import json

async def upload_documents_batch(documents, jwt_token):
    """Upload multiple documents concurrently"""
    
    async def upload_single(session, doc_path, metadata):
        data = aiohttp.FormData()
        data.add_field('file', open(doc_path, 'rb'))
        data.add_field('metadata', json.dumps(metadata))
        
        headers = {'Authorization': f'Bearer {jwt_token}'}
        
        async with session.post(
            'http://localhost:6000/api/v1/categorise',
            data=data,
            headers=headers
        ) as response:
            return await response.json()
    
    async with aiohttp.ClientSession() as session:
        tasks = [
            upload_single(session, doc['path'], doc['metadata'])
            for doc in documents
        ]
        results = await asyncio.gather(*tasks)
        return results

# Usage
documents = [
    {'path': 'motion1.pdf', 'metadata': {'case_id': '2024-CV-001'}},
    {'path': 'motion2.pdf', 'metadata': {'case_id': '2024-CV-002'}},
    {'path': 'brief1.pdf', 'metadata': {'case_id': '2024-CV-003'}}
]

results = asyncio.run(upload_documents_batch(documents, jwt_token))
```

### Error Recovery Patterns
```javascript
class MotionIndexClient {
  constructor(baseUrl, authToken) {
    this.baseUrl = baseUrl;
    this.authToken = authToken;
    this.retryConfig = {
      maxRetries: 3,
      backoffFactor: 2,
      initialDelay: 1000
    };
  }

  async searchWithRetry(searchRequest) {
    let lastError;
    
    for (let attempt = 0; attempt < this.retryConfig.maxRetries; attempt++) {
      try {
        const response = await fetch(`${this.baseUrl}/api/v1/search`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(searchRequest)
        });

        if (response.ok) {
          return await response.json();
        }

        // Don't retry client errors (4xx)
        if (response.status >= 400 && response.status < 500) {
          throw new Error(`Client error: ${response.status}`);
        }

        lastError = new Error(`Server error: ${response.status}`);
      } catch (error) {
        lastError = error;
        
        if (attempt < this.retryConfig.maxRetries - 1) {
          const delay = this.retryConfig.initialDelay * 
                       Math.pow(this.retryConfig.backoffFactor, attempt);
          await new Promise(resolve => setTimeout(resolve, delay));
        }
      }
    }

    throw new Error(`Search failed after ${this.retryConfig.maxRetries} attempts: ${lastError.message}`);
  }

  async uploadWithProgressTracking(file, metadata, onProgress) {
    return new Promise((resolve, reject) => {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('metadata', JSON.stringify(metadata));

      const xhr = new XMLHttpRequest();

      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable) {
          const percentComplete = (event.loaded / event.total) * 100;
          onProgress(percentComplete);
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status === 200) {
          resolve(JSON.parse(xhr.responseText));
        } else {
          reject(new Error(`Upload failed: ${xhr.status}`));
        }
      });

      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed: Network error'));
      });

      xhr.open('POST', `${this.baseUrl}/api/v1/categorise`);
      xhr.setRequestHeader('Authorization', `Bearer ${this.authToken}`);
      xhr.send(formData);
    });
  }
}
```

## Testing & Development

### Testing with Mock Data
```bash
# Test health endpoints
curl -f http://localhost:6000/health || echo "Health check failed"
curl -f http://localhost:6000/health/detailed || echo "Detailed health check failed"

# Test search with sample query
curl -X POST http://localhost:6000/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "test", "limit": 1}' \
  --fail --silent --show-error

# Test file upload with sample document
echo "Sample document content" > test.txt
curl -X POST http://localhost:6000/api/v1/categorise \
  -F "file=@test.txt" \
  -F "metadata={\"test\": true}" \
  --fail --silent --show-error
rm test.txt
```

### Local Development Setup
```bash
# 1. Clone repository
git clone https://github.com/your-org/motion-index-fiber
cd motion-index-fiber

# 2. Install dependencies
go mod tidy

# 3. Set up environment
cp .env.example .env
# Edit .env with your configuration

# 4. Run development server
go run cmd/server/main.go

# 5. Test API
curl http://localhost:6000/health
```

### API Testing Scripts
```python
#!/usr/bin/env python3
"""
API integration test suite
"""
import requests
import json
import sys
import time

BASE_URL = "http://localhost:6000"

def test_health():
    """Test health endpoints"""
    print("Testing health endpoints...")
    
    # Basic health
    resp = requests.get(f"{BASE_URL}/health")
    assert resp.status_code == 200
    assert resp.json()["success"] == True
    
    # Detailed health
    resp = requests.get(f"{BASE_URL}/health/detailed")
    assert resp.status_code == 200
    assert "storage" in resp.json()["data"]
    
    print("✓ Health checks passed")

def test_search():
    """Test search functionality"""
    print("Testing search...")
    
    search_payload = {
        "query": "test",
        "limit": 5
    }
    
    resp = requests.post(f"{BASE_URL}/api/v1/search", json=search_payload)
    assert resp.status_code == 200
    
    data = resp.json()
    assert data["success"] == True
    assert "total" in data["data"]
    
    print("✓ Search test passed")

def test_document_upload():
    """Test document upload"""
    print("Testing document upload...")
    
    # Create test file
    test_content = "This is a test legal document for API testing."
    files = {"file": ("test.txt", test_content, "text/plain")}
    data = {
        "metadata": json.dumps({
            "test": True,
            "case_id": "TEST-001"
        })
    }
    
    resp = requests.post(f"{BASE_URL}/api/v1/categorise", files=files, data=data)
    assert resp.status_code == 200
    
    result = resp.json()
    assert result["success"] == True
    assert "document_id" in result["data"]
    
    print("✓ Document upload test passed")

def run_tests():
    """Run all tests"""
    print(f"Running API tests against {BASE_URL}")
    print("=" * 50)
    
    try:
        test_health()
        test_search()
        test_document_upload()
        
        print("=" * 50)
        print("✓ All tests passed!")
        return True
        
    except Exception as e:
        print(f"✗ Test failed: {e}")
        return False

if __name__ == "__main__":
    success = run_tests()
    sys.exit(0 if success else 1)
```

## Performance Optimization

### Caching Strategies
```javascript
// Client-side caching for search results
class CachedMotionIndexClient {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
    this.searchCache = new Map();
    this.cacheTimeout = 5 * 60 * 1000; // 5 minutes
  }

  async search(searchRequest) {
    const cacheKey = JSON.stringify(searchRequest);
    const cached = this.searchCache.get(cacheKey);
    
    if (cached && Date.now() - cached.timestamp < this.cacheTimeout) {
      return cached.data;
    }

    const response = await fetch(`${this.baseUrl}/api/v1/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(searchRequest)
    });

    const data = await response.json();
    
    this.searchCache.set(cacheKey, {
      data: data,
      timestamp: Date.now()
    });

    return data;
  }
}
```

### Pagination Best Practices
```python
def paginated_search(query, total_results_needed=1000):
    """Efficiently retrieve large result sets"""
    all_documents = []
    page_size = 50  # Optimal page size
    offset = 0
    
    while len(all_documents) < total_results_needed:
        search_request = {
            "query": query,
            "limit": page_size,
            "offset": offset
        }
        
        response = requests.post("/api/v1/search", json=search_request)
        data = response.json()
        
        if not data["success"] or not data["data"]["documents"]:
            break
            
        all_documents.extend(data["data"]["documents"])
        
        # Check if we've reached the end
        if len(data["data"]["documents"]) < page_size:
            break
            
        offset += page_size
        
        # Rate limiting - be respectful
        time.sleep(0.1)
    
    return all_documents[:total_results_needed]
```

## Security Best Practices

### Token Management
```javascript
// Secure token storage and refresh
class SecureTokenManager {
  constructor() {
    this.token = null;
    this.refreshToken = null;
    this.tokenExpiry = null;
  }

  setTokens(accessToken, refreshToken, expiresIn) {
    this.token = accessToken;
    this.refreshToken = refreshToken;
    this.tokenExpiry = Date.now() + (expiresIn * 1000);
    
    // Store refresh token securely (httpOnly cookie in real app)
    localStorage.setItem('refresh_token', refreshToken);
  }

  async getValidToken() {
    if (!this.token || Date.now() >= this.tokenExpiry - 60000) {
      await this.refreshAccessToken();
    }
    return this.token;
  }

  async refreshAccessToken() {
    const refreshToken = localStorage.getItem('refresh_token');
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }

    const response = await fetch('/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken })
    });

    if (!response.ok) {
      throw new Error('Token refresh failed');
    }

    const data = await response.json();
    this.setTokens(data.access_token, data.refresh_token, data.expires_in);
  }
}
```

### Input Validation
```python
def validate_search_request(request_data):
    """Validate search request parameters"""
    errors = []
    
    # Validate query
    query = request_data.get('query', '')
    if len(query) > 1000:
        errors.append('Query too long (max 1000 characters)')
    
    # Validate limit
    limit = request_data.get('limit', 20)
    if not isinstance(limit, int) or limit < 1 or limit > 100:
        errors.append('Limit must be between 1 and 100')
    
    # Validate offset  
    offset = request_data.get('offset', 0)
    if not isinstance(offset, int) or offset < 0:
        errors.append('Offset must be non-negative')
    
    # Validate filters
    filters = request_data.get('filters', {})
    if not isinstance(filters, dict):
        errors.append('Filters must be an object')
    
    if errors:
        raise ValueError(f"Validation failed: {', '.join(errors)}")
    
    return True
```

## Support

- **Documentation Issues**: Open an issue on GitHub
- **API Questions**: See [FAQ](./faq.md) 
- **Feature Requests**: Use GitHub discussions
- **Security Issues**: Contact security@motionindex.com
- **Performance Issues**: Include request/response details in bug reports
- **Integration Help**: Check [Integration Guide](./integration.md)