# Stage 1: Build
FROM golang:1.24.6-alpine AS builder

RUN apk add --no-cache \
    git \
    build-base \
    tesseract-ocr-dev \
    poppler-dev \
    cairo-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ENV CGO_ENABLED=1
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./cmd/server/main.go

# Stage 2: Runtime
FROM alpine:3.19

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
    wget

WORKDIR /app

COPY --from=builder /app/server /app/server

ENV PORT=8003
ENV TESSDATA_PREFIX=/usr/share/tessdata

EXPOSE 8003

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

CMD ["/app/server"]
