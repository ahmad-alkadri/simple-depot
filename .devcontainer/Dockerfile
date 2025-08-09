FROM golang:1.24.4-slim
RUN apt-get update && apt-get install -y git curl \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
# Optional: install golangci-lint for linting
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# Create non-root user for security
RUN useradd -m dev && chown -R dev /workspace
USER dev
