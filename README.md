# Simple Depot

**simple-depot** is a lightweight, extensible Go HTTP server for capturing, storing, and inspecting incoming HTTP request payloads. It supports JSON, multipart form data, and arbitrary binary data, saving them locally or to a MinIO/S3-compatible backend. Designed for debugging, prototyping, and webhook testing.

---

## Table of Contents
1. [File Structure](#file-structure)
2. [Features](#features)
3. [Setup & Development Environment](#setup--development-environment)
4. [Configuration](#configuration)
5. [Launching the Server](#launching-the-server)
6. [API Usage](#api-usage)
7. [Listing & Retrieving Payloads](#listing--retrieving-payloads)
8. [Extending & Customizing](#extending--customizing)
9. [Future Enhancements](#future-enhancements)
10. [License](#license)

---

## File Structure

```
simple-depot/
├── cmd/                 # Entry points (API server)
├── internal/
│   ├── config/          # Configuration management
│   ├── handlers/        # HTTP handlers (Depot, List, Get)
│   ├── http/            # HTTP utilities
│   ├── middleware/      # Middleware (auth, logging, etc.)
│   ├── payload/         # Payload processing logic
│   ├── services/        # Service interfaces & implementations
│   ├── storage/         # Storage backends (local, MinIO)
├── pkg/                 # Shared utilities
├── testdata/            # Sample payloads for testing
├── tests/               # Unit & integration tests
├── tmp/                 # Default local payload storage
├── main.go              # Main server entry point
├── go.mod, go.sum       # Go modules
├── LICENSE              # MIT License
├── README.md            # This file
└── ...
```

---

## Features

- **HTTP POST endpoint** at `/depot` for capturing payloads
- **Supports**: JSON, multipart file uploads, arbitrary binary data
- **Asynchronous storage** to local `tmp/` or MinIO/S3
- **List & retrieve** stored payloads via `/list` and `/get`
- **Zero dependencies** (uses Go standard library)
- **Extensible**: add storage backends, UI, metadata, authentication

---

## Setup & Development Environment

### Prerequisites

To set up the development environment, ensure you have the following tools installed:

- **Go** (version 1.18 or later)
- **Git** (for cloning the repository)
- **curl** (for sending HTTP requests to test endpoints)
- **VS Code** (recommended for development)
- **Docker** (optional, for running in containers or using dev containers)
- **MinIO** (optional, if you want to test S3-compatible storage backend)
- **make** (optional, for build/test automation)

### Clone & Install

```bash
git clone https://github.com/ahmad-alkadri/simple-depot.git
cd simple-depot
go mod tidy
```

---

## Configuration

- **Default port**: `3003`
- **Default storage**: `./tmp` (local directory)
- **MinIO/S3 support**: Configure in `main.go` or via `internal/config/config.go`
- **Customizing**: Change port, storage backend, or other settings in config files or code.

---

## Launching the Server

### Direct Run
```bash
go run main.go
```

### Build & Run
```bash
go build -o simple-depot main.go
./simple-depot
```

You should see:
```
Server listening on :3003
```

---

## API Usage

### 1. Capture Payload (`POST /depot`)

Send requests to `http://localhost:3003/depot`:

- **JSON**
  ```bash
  curl -X POST \
    -H "Content-Type: application/json" \
    -d '{"message":"Hello World"}' \
    http://localhost:3003/depot
  ```

- **Multipart File Upload**
  ```bash
  curl -X POST \
    -F "file=@/path/to/local-file.txt" \
    http://localhost:3003/depot
  ```

- **Binary Data**
  ```bash
  curl -X POST \
    --data-binary @/path/to/image.png \
    -H "Content-Type: application/octet-stream" \
    http://localhost:3003/depot
  ```

**Response:**
Returns JSON with request ID, payload size, timestamp, and filename.

### 2. List All Payloads (`GET /list`)

```bash
curl -X GET http://localhost:3003/list
```
Returns a JSON array of stored payloads and their metadata.

### 3. Retrieve Payload (`GET /get?request_id=<id>&raw=true|false`)

```bash
curl -X GET "http://localhost:3003/get?request_id=<id>&raw=true"
```
- If `raw=true`, returns the file (or zip if multiple files) as a download.
- If `raw=false` (default), returns JSON metadata and base64-encoded payload.

---

## Output & Storage

- **JSON bodies**: `tmp/<timestamp>.json`
- **Multipart files**: `tmp/<original_filename>`
- **Binary data**: `tmp/<timestamp>.bin`
- **MinIO/S3**: If configured, files are stored in the specified bucket.

Logs include request timestamp, payload size, and file paths.

---

## Extending & Customizing

- Add new storage backends in `internal/storage/`
- Implement authentication in `internal/middleware/`
- Add metadata extraction in `internal/payload/`
- UI/web frontend can be added for browsing payloads

---

## Future Enhancements

- Database/cloud storage backends (AWS S3, etc.)
- Web UI for browsing requests
- Configurable endpoints & storage locations
- Authentication & access control
- Structured metadata (headers, query params, IP)

---

## License

MIT License. See [LICENSE](LICENSE).

