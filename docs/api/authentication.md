# Authentication

Motion Index API uses JSON Web Tokens (JWT) for authentication. All protected endpoints require a valid JWT token in the `Authorization` header.

## Authentication Flow

### 1. Token Acquisition
Tokens are issued by **Supabase** and obtained through the client application's authentication flow.

```bash
# Authenticate with Supabase (example)
curl -X POST https://your-project.supabase.co/auth/v1/token \
  -H "Content-Type: application/json" \
  -H "apikey: your-supabase-anon-key" \
  -d '{
    "email": "user@example.com",
    "password": "password"
  }'
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer",
  "expires_in": 3600,
  "refresh_token": "...",
  "user": {
    "id": "user-uuid",
    "email": "user@example.com"
  }
}
```

### 2. Token Usage
Include the token in the `Authorization` header for all protected endpoints:

```bash
curl -X POST http://localhost:6000/api/v1/update-metadata \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"document_id": "doc_123", "metadata": {...}}'
```

## JWT Token Structure

### Header
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

### Payload
```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "role": "authenticated",
  "iat": 1640995200,
  "exp": 1640998800,
  "aud": "authenticated",
  "iss": "https://your-project.supabase.co/auth/v1"
}
```

## Protected Endpoints

### Administrative Operations
- `POST /api/v1/update-metadata` - Update document metadata
- `DELETE /api/v1/documents/:id` - Delete documents

### Future Protected Endpoints
- `POST /api/v1/users/profile` - Update user profile
- `GET /api/v1/users/documents` - Get user's documents
- `POST /api/v1/admin/*` - Administrative functions

## Authentication Errors

### 401 Unauthorized - Missing Token
```json
{
  "success": false,
  "error": {
    "code": "AUTHENTICATION_ERROR",
    "message": "Authorization header is required",
    "details": {
      "required_format": "Bearer <token>",
      "header_name": "Authorization"
    }
  }
}
```

### 401 Unauthorized - Invalid Token Format
```json
{
  "success": false,
  "error": {
    "code": "AUTHENTICATION_ERROR",
    "message": "Invalid authorization header format",
    "details": {
      "provided_format": "Token abc123",
      "required_format": "Bearer <token>",
      "issue": "Missing 'Bearer' prefix"
    }
  }
}
```

### 401 Unauthorized - Invalid Token
```json
{
  "success": false,
  "error": {
    "code": "AUTHENTICATION_ERROR",
    "message": "Invalid or expired JWT token",
    "details": {
      "token_expired": true,
      "expires_at": "2024-01-01T12:00:00Z",
      "issued_at": "2024-01-01T11:00:00Z",
      "current_time": "2024-01-01T12:30:00Z"
    }
  }
}
```

### 401 Unauthorized - Malformed Token
```json
{
  "success": false,
  "error": {
    "code": "AUTHENTICATION_ERROR",
    "message": "Malformed JWT token",
    "details": {
      "issue": "Token structure is invalid",
      "expected_parts": 3,
      "actual_parts": 2,
      "hint": "JWT tokens should have 3 parts separated by dots"
    }
  }
}
```

### 403 Forbidden - Insufficient Permissions
```json
{
  "success": false,
  "error": {
    "code": "AUTHORIZATION_ERROR",
    "message": "Insufficient permissions",
    "details": {
      "required_role": "admin",
      "user_role": "user",
      "action": "delete_document",
      "resource": "documents/doc_123"
    }
  }
}
```

## Token Validation Process

The API validates JWT tokens through the following steps:

1. **Header Validation**: Checks for `Authorization: Bearer <token>` format
2. **Token Structure**: Validates JWT has 3 parts (header.payload.signature)
3. **Signature Verification**: Verifies token signature using Supabase secret
4. **Expiration Check**: Ensures token hasn't expired
5. **Issuer Validation**: Confirms token was issued by trusted Supabase instance
6. **Role Extraction**: Extracts user role and permissions from token claims

## Security Best Practices

### Client-Side
- **Never store tokens in localStorage** - Use secure, httpOnly cookies when possible
- **Implement token refresh** - Refresh tokens before expiration
- **Clear tokens on logout** - Remove all authentication artifacts
- **Use HTTPS only** - Never send tokens over unencrypted connections

### Token Handling
- **Short expiration times** - Default 1 hour, maximum 24 hours
- **Automatic refresh** - Implement silent token refresh
- **Logout invalidation** - Invalidate tokens on logout (Supabase handles this)

### Error Handling
- **Don't expose sensitive details** - Authentication errors should be generic
- **Log security events** - Log failed authentication attempts
- **Rate limiting** - Implement rate limiting on auth endpoints

## Implementation Examples

### JavaScript/TypeScript (Supabase Client)
```javascript
import { createClient } from '@supabase/supabase-js'

const supabase = createClient(
  'https://your-project.supabase.co',
  'your-supabase-anon-key'
)

// Login
const { data, error } = await supabase.auth.signInWithPassword({
  email: 'user@example.com',
  password: 'password'
})

// Use token for API calls
const token = data.session?.access_token
const response = await fetch('/api/v1/update-metadata', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ document_id: 'doc_123', metadata: {} })
})
```

### Python
```python
import requests
import json

# Get token from Supabase (example)
auth_response = requests.post(
    'https://your-project.supabase.co/auth/v1/token',
    headers={
        'Content-Type': 'application/json',
        'apikey': 'your-supabase-anon-key'
    },
    json={
        'email': 'user@example.com',
        'password': 'password'
    }
)

token = auth_response.json()['access_token']

# Use token for API calls
response = requests.post(
    'http://localhost:6000/api/v1/update-metadata',
    headers={
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/json'
    },
    json={
        'document_id': 'doc_123',
        'metadata': {}
    }
)
```

### cURL
```bash
# Store token in variable
TOKEN=$(curl -s -X POST https://your-project.supabase.co/auth/v1/token \
  -H "Content-Type: application/json" \
  -H "apikey: your-supabase-anon-key" \
  -d '{"email":"user@example.com","password":"password"}' \
  | jq -r '.access_token')

# Use token for API calls
curl -X POST http://localhost:6000/api/v1/update-metadata \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"document_id":"doc_123","metadata":{}}'
```

## Testing Authentication

### Postman Setup
1. Create environment variables:
   - `supabase_url`: Your Supabase project URL
   - `supabase_anon_key`: Your Supabase anonymous key
   - `api_base_url`: Your API base URL

2. Add pre-request script for automatic token refresh:
```javascript
// Pre-request script
const authUrl = pm.environment.get("supabase_url") + "/auth/v1/token";
const apiKey = pm.environment.get("supabase_anon_key");

pm.sendRequest({
    url: authUrl,
    method: 'POST',
    header: {
        'Content-Type': 'application/json',
        'apikey': apiKey
    },
    body: {
        mode: 'raw',
        raw: JSON.stringify({
            email: "test@example.com",
            password: "testpassword"
        })
    }
}, function (err, response) {
    if (!err && response.json().access_token) {
        pm.environment.set("access_token", response.json().access_token);
    }
});
```

3. Use token in requests:
```
Authorization: Bearer {{access_token}}
```

## Troubleshooting

### Common Issues

#### "Authorization header is required"
- **Cause**: Missing `Authorization` header
- **Solution**: Add `Authorization: Bearer <token>` header

#### "Invalid authorization header format"
- **Cause**: Incorrect header format (e.g., `Token abc123` instead of `Bearer abc123`)
- **Solution**: Use correct format: `Bearer <token>`

#### "Invalid or expired JWT token"
- **Cause**: Token has expired or is invalid
- **Solution**: Refresh token or re-authenticate

#### "Malformed JWT token"
- **Cause**: Token structure is corrupted
- **Solution**: Check token has 3 parts separated by dots

### Debug Mode
For development, you can enable authentication debugging by setting:
```bash
DEBUG_AUTH=true
```

This will provide detailed authentication logs (never use in production).

## Rate Limiting

Authentication endpoints are rate limited:
- **Login attempts**: 5 per minute per IP
- **Token refresh**: 10 per minute per user
- **Protected endpoints**: 100 per minute per authenticated user

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
X-RateLimit-Window: 60
```