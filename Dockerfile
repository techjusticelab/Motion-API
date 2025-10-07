# Multi-stage build for Motion-Index Fiber - DigitalOcean App Platform optimized
# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    pkgconfig \
    tesseract-ocr-dev \
    poppler-dev \
    cairo-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -extldflags '-static'" \
    -a -installsuffix cgo \
    -o server \
    ./cmd/server/main.go

# Stage 2: Runtime stage
FROM alpine:3.19

# Install runtime dependencies for legal document processing
RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    poppler-utils \
    imagemagick \
    fontconfig \
    ttf-dejavu \
    ttf-liberation \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN adduser -D -s /bin/sh -u 1000 appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/server .

# Create necessary directories and set permissions
RUN mkdir -p /app/data /app/logs && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Configure Tesseract for legal document OCR
ENV TESSDATA_PREFIX=/usr/share/tessdata

# App Platform expects the application to listen on the PORT environment variable
ENV PORT=8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

# Expose port (App Platform will override this)
EXPOSE 8080

# Run the application
CMD ["./server"]