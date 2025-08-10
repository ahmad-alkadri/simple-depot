package main

import (
	"encoding/base64"
)

// DefaultResponseFormatter handles formatting HTTP responses
type DefaultResponseFormatter struct{}

// NewDefaultResponseFormatter creates a new response formatter
func NewDefaultResponseFormatter() *DefaultResponseFormatter {
	return &DefaultResponseFormatter{}
}

// FormatDepotResponse formats the response for depot endpoint
func (f *DefaultResponseFormatter) FormatDepotResponse(requestID string, size int, timestamp string, filename string) map[string]any {
	response := map[string]any{
		"status":     "accepted",
		"request_id": requestID,
		"size":       size,
		"timestamp":  timestamp,
	}

	if filename != "" {
		response["original_filename"] = filename
	}

	return response
}

// FormatGetResponse formats the response for get endpoint
func (f *DefaultResponseFormatter) FormatGetResponse(requestID string, files []FileInfo, count int) map[string]any {
	return map[string]any{
		"request_id": requestID,
		"files":      files,
		"count":      count,
	}
}

// FormatListResponse formats the response for list endpoint
func (f *DefaultResponseFormatter) FormatListResponse(objects []string, count int) map[string]any {
	return map[string]any{
		"count":   count,
		"objects": objects,
	}
}

// FormatFileInfo creates a FileInfo struct from payload data
func (f *DefaultResponseFormatter) FormatFileInfo(objectName, originalFilename string, data []byte, contentType string) FileInfo {
	return FileInfo{
		ObjectName:       objectName,
		OriginalFilename: originalFilename,
		Size:             len(data),
		ContentType:      contentType,
		PayloadBase64:    base64.StdEncoding.EncodeToString(data),
	}
}
