# Multi-stage Dockerfile for Motion Index Fiber API

FROM golang:1.24

WORKDIR /app

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# App Platform sets PORT at runtime; expose a default for local runs
EXPOSE 8080

# Simple: compile-and-run at container start
CMD ["go", "run", "cmd/server/main.go"]
