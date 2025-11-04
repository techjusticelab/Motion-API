# Process Mass Upload

This directory contains utilities for processing and uploading large batches of files to the Motion Index API.

## Files

- `api-helpers/` - Shared HTTP helpers for upload, conversion, and extraction requests
- `clean_unsupported_file_types.go` - Removes files with unsupported extensions
- `delete_duplicate_files.go` - Removes duplicate files based on content hash
- `main.go` - CLI command dispatcher (clean, dedupe, upload)

## Upload Command (`go run main.go upload`)

The upload command sends supported files to the API's `/api/v1/upload/s3` endpoint.

### Key Features

1. **File Discovery**: Automatically finds all valid file types (.pdf, .doc, .docx, .ppt, .pptx, .txt) in the specified directory and subdirectories
2. **Concurrent Uploads**: Configurable worker pool (default 5 concurrent uploads)
3. **Retry Logic**: Automatic retries with configurable retry count and delay
4. **Progress Tracking**: Shows per-file progress with success/failure status
5. **Comprehensive Error Handling**: Collects errors and prints a detailed summary at the end
6. **API Integration**: Uses the shared HTTP helpers under `api-helpers/` to format multipart requests

### Usage

```bash
# Navigate to the directory
cd /home/okita/Scripts/Work/TJL/motion-api/data/process-mass-upload

# Upload files from current directory
go run main.go upload

# Upload files from a specific directory
go run main.go upload --dir ../Articles-20251029T211736Z-1-001/

# Override the upload endpoint
go run main.go upload --endpoint http://localhost:8003/api/v1/upload/s3

# Limit uploads to 40MB files
go run main.go upload --max-size 40MB ./some/path

# Show all available options
go run main.go upload -h
```

### Configuration

Default values can be overridden with flags or environment variables:

- **Endpoint**: `--endpoint` flag or `MASS_UPLOAD_ENDPOINT` / `MASS_UPLOAD_API_BASE_URL`
- **Concurrency**: `--concurrency` flag or `MASS_UPLOAD_CONCURRENCY` (default `5`)
- **Retries**: `--retries` flag or `MASS_UPLOAD_RETRIES` (default `3`)
- **Retry Delay**: `--retry-delay` flag or `MASS_UPLOAD_RETRY_DELAY` (default `2s`)
- **Timeout**: `--timeout` flag or `MASS_UPLOAD_TIMEOUT` (default `5m`)
- **Maximum File Size**: `--max-size` flag or `MASS_UPLOAD_MAX_SIZE` (default `100MB`)

Environment values accept either Go duration strings (`2s`, `1m30s`) or integer seconds.
File sizes accept raw bytes (`104857600`) or values with suffixes (`100MB`, `80M`, `512KB`).

### Supported File Types

Only files with these extensions are uploaded:

- `.pdf` - PDF documents
- `.doc` - Microsoft Word documents (legacy)
- `.docx` - Microsoft Word documents (modern)
- `.ppt` - Microsoft PowerPoint presentations (legacy)
- `.pptx` - Microsoft PowerPoint presentations (modern)
- `.txt` - Plain text files

Unsupported files are skipped automatically and summarized at the end of the run.

### Output

The command prints:

- Per-file progress with success or failure markers
- Retry notifications when attempts are retried
- A final summary including:
  - Total files processed
  - Number of successful uploads
  - Number of failed uploads
  - Number of skipped (unsupported) files
  - Total retry attempts
  - Success percentage
  - Total duration
  - Detailed list of failed files (if any)

### Example Output

```
Starting upload of files to API from: /home/okita/Scripts/Work/TJL/motion-api/data/Articles-20251029T211736Z-1-001
API Endpoint: http://localhost:8003/api/v1/upload/s3
-------------------------------------------------------
Found 150 supported files to upload (5 skipped as unsupported) (2 skipped for size > 100.0MB)
Max concurrency: 5 | Retries per file: 3 | Timeout: 5m0s
-------------------------------------------------------
[1/150] ✓ Uploaded: document1.pdf
[2/150] ✓ Uploaded: presentation.pptx
↻ Retry 1/3 for large_file.pdf: upload failed with status 413: Request Entity Too Large
[3/150] ✗ Failed: large_file.pdf - upload failed with status 413: Request Entity Too Large
[4/150] ✓ Uploaded: report.docx
...
-------------------------------------------------------
Upload Summary:
  Total files: 150
  Successful: 148
  Failed: 2
  Skipped (unsupported): 5
  Total retries: 3
  Success rate: 98.7%
  Duration: 3m12s

Failed uploads:
  ✗ large_file.pdf: upload failed with status 413: Request Entity Too Large
  ✗ corrupted.doc: failed to open file: permission denied

Skipped files (unsupported extensions):
  - archive.zip
  - notes.md
  ... and 3 more

Skipped files (exceeded max size):
  - scan.pdf (115.2MB)
  - deposition.mov (250.0MB)
-------------------------------------------------------
```

### Error Handling

The upload command handles:

- **File access errors**: Permission denied, file not found
- **Network errors**: Connection timeouts, DNS failures
- **API errors**: HTTP status codes and server responses
- **File format errors**: Unsupported file types are skipped automatically
- **Size limit errors**: Files exceeding the configured `--max-size` (default 100MB) are filtered locally

Failed uploads are retried according to the configured retry count and delay.

## Prerequisites

1. **Go Runtime**: Ensure Go is installed and available in your PATH
2. **API Server**: The Motion Index API server should be running on `localhost:8003`
3. **File Permissions**: Read access to the files you want to upload
4. **Network Access**: Connection to the API server

## Workflow

For processing a large batch of files, follow this workflow:

1. **Clean unsupported files** (optional):
   ```bash
   go run clean_unsupported_file_types.go ../
   ```

2. **Remove duplicates** (optional):
   ```bash
   go run delete_duplicate_files.go ../
   ```

3. **Upload to API**:
   ```bash
   go run main.go upload ../
   ```

## Troubleshooting

### Common Issues

1. **Connection refused**: Ensure the API server is running on port 8003
2. **File too large**: Reduce `--max-size` (default 100MB) or split oversized files
3. **Permission denied**: Check file permissions and directory access
4. **Out of memory**: Large number of concurrent uploads may require reducing `--concurrency`

### Debug Tips

- Check the API server logs for detailed error information
- Verify file extensions match the supported types
- Ensure sufficient disk space for temporary operations
- Monitor network connectivity during uploads

## API Integration

The upload command integrates with the Motion Index API's upload endpoint:

- **Endpoint**: `POST /api/v1/upload/s3`
- **Content-Type**: `multipart/form-data`
- **File Field**: `file`
- **Response**: JSON with upload details or error information

The uploaded files are stored in the `unprocessed/` directory in DigitalOcean Spaces and can be processed by the API's document processing pipeline.