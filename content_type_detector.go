package main

import (
	"mime"
	"path/filepath"
	"strings"
)

// DefaultContentTypeDetector detects content types from various sources
type DefaultContentTypeDetector struct{}

// NewDefaultContentTypeDetector creates a new content type detector
func NewDefaultContentTypeDetector() *DefaultContentTypeDetector {
	return &DefaultContentTypeDetector{}
}

// DetectFromData detects content type from raw data (basic implementation)
func (d *DefaultContentTypeDetector) DetectFromData(data []byte) string {
	// Basic detection based on first few bytes
	if len(data) == 0 {
		return "application/octet-stream"
	}

	// Check for JSON
	if len(data) > 0 && (data[0] == '{' || data[0] == '[') {
		return "application/json"
	}

	// Check for common image formats
	if len(data) >= 4 {
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			return "image/jpeg"
		}
		if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
			return "image/png"
		}
	}

	return "application/octet-stream"
}

// DetectFromFilename detects content type from filename extension
func (d *DefaultContentTypeDetector) DetectFromFilename(filename string) string {
	if filename == "" {
		return "application/octet-stream"
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return "application/json"
	case ".txt":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	default:
		return "application/octet-stream"
	}
}

// DetectFromContentType processes and normalizes existing content type
func (d *DefaultContentTypeDetector) DetectFromContentType(contentType string) string {
	if contentType == "" {
		return "application/octet-stream"
	}

	// Parse media type to handle parameters like charset
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "application/octet-stream"
	}

	return mediaType
}
