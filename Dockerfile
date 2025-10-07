FROM golang:1.24.6-alpine

# Install runtime dependencies for legal document processing
RUN apk add --no-cache \
    git \
    build-base \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    poppler-utils \
    imagemagick \
    fontconfig \
    ttf-dejavu \
    ttf-liberation \
    ca-certificates \
    tzdata \
    wget

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source
COPY . .

# App Platform expects the app to listen on $PORT
ENV PORT=8003
ENV TESSDATA_PREFIX=/usr/share/tessdata

EXPOSE 8003

# Simple healthcheck
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

# Run via go run for simplicity as requested
CMD ["go", "run", "cmd/server/main.go"]
