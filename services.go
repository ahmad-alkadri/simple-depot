package main

// PayloadProcessor handles processing different types of payloads
type PayloadProcessor interface {
	Process(requestID string, data []byte, contentType string, filename string) ([]ProcessedPayload, error)
}

// ProcessedPayload represents a processed payload ready for storage
type ProcessedPayload struct {
	ObjectName  string
	Data        []byte
	ContentType string
	Filename    string
}

// IDGenerator generates unique identifiers
type IDGenerator interface {
	Generate() string
}

// ContentTypeDetector detects content types from data or filenames
type ContentTypeDetector interface {
	DetectFromData(data []byte) string
	DetectFromFilename(filename string) string
	DetectFromContentType(contentType string) string
}

// FilenameExtractor extracts filenames from HTTP requests
type FilenameExtractor interface {
	Extract(contentDisposition string) string
}

// ResponseFormatter formats HTTP responses
type ResponseFormatter interface {
	FormatDepotResponse(requestID string, size int, timestamp string, filename string) map[string]any
	FormatGetResponse(requestID string, files []FileInfo, count int) map[string]any
	FormatListResponse(objects []string, count int) map[string]any
	FormatFileInfo(objectName, originalFilename string, data []byte, contentType string) FileInfo
}

// FileInfo represents file information for responses
type FileInfo struct {
	ObjectName       string `json:"object_name"`
	OriginalFilename string `json:"original_filename"`
	Size             int    `json:"size"`
	ContentType      string `json:"content_type"`
	PayloadBase64    string `json:"payload_base64"`
}

// ZipService handles creating zip archives
type ZipService interface {
	CreateZip(files []FileInfo) ([]byte, error)
}

// PayloadService orchestrates payload operations
type PayloadService interface {
	StorePayload(data []byte, contentType string, filename string) (string, error)
	RetrievePayloads(requestID string, raw bool) (interface{}, error)
	ListAllPayloads() ([]string, error)
}
