FROM golang:1.24.6-alpine

# Install runtime deps required by the app
RUN apk add --no-cache \
    git \
    build-base \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    poppler-utils \
    imagemagick \
    ca-certificates \
    tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ENV PORT=8003
EXPOSE 8003

CMD ["go", "run", "cmd/server/main.go"]
