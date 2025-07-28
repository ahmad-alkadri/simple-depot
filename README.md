# simple-depot

simple-depot is a lightweight Go HTTP server for capturing and saving incoming request payloads (JSON, multipart form data, or binary). It listens on `/depot`, processes requests asynchronously, and stores payloads locally in the `tmp` directory.

Because of its local nature, this tool is not designed for production-level deployment (yet).

## Features
 - HTTP POST endpoint at `/depot`
 - Supports JSON, multipart file uploads, and arbitrary binary data
 - Asynchronous storage to `./tmp`
 - Zero dependencies (standard library only)
 - Extensible: storage backends, UI, metadata

## Getting Started

### Prerequisites

- Go 1.18 or later installed on your system
- Git (for cloning the repository)

### Installation

Clone the repository and install dependencies:
```bash
git clone https://github.com/ahmad-alkadri/simple-depot.git
cd simple-depot
go mod tidy
```

### Configuration

- By default, the server listens on port `3003` and saves payloads under `./tmp`.
- You can customize these values by modifying the code in `main.go` as needed.

## Usage

To start the server:

```bash
## Direct run
go run main.go

## Build and run
go build -o simple-depot main.go
./simple-depot
```

You should see a log entry indicating the server is listening:

```
Server listening on :3003
```

### Sending Requests

Once the server is running, send HTTP requests to `http://localhost:3003/depot`:

- JSON payload:
  ```bash
  curl -X POST \
    -H "Content-Type: application/json" \
    -d '{"message":"Hello World"}' \
    http://localhost:3003/depot
  ```

- Multipart form upload:
  ```bash
  curl -X POST \
    -F "file=@/path/to/local-file.txt" \
    http://localhost:3003/depot
  ```

- Arbitrary binary data:
  ```bash
  curl -X POST \
    --data-binary @/path/to/image.png \
    -H "Content-Type: application/octet-stream" \
    http://localhost:3003/depot
  ```

### Output

- JSON bodies are saved as `tmp/<timestamp>.json`
- Multipart file uploads are saved with their original filenames in `tmp/`
- Other payloads are stored as `tmp/<timestamp>.bin`

Logs will include details such as the request timestamp, payload size, and saved file paths.

## Future Enhancements

- Support for database or cloud object storage backends (e.g., AWS S3)
- Web UI to browse captured requests
- Configurable endpoint paths and storage locations
- Authentication and access control
- Structured metadata storage (headers, query parameters, IP address)

## License

This project is released under the MIT License. See [LICENSE](LICENSE) for details.

---

*simple-depot* helps you quickly capture, inspect, and store HTTP request payloads for debugging, prototyping, or webhook testing.
