package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DefaultPayloadProcessor handles processing different types of payloads
type DefaultPayloadProcessor struct {
	contentTypeDetector ContentTypeDetector
	multipartProcessor  *MultipartPayloadProcessor
}

// NewDefaultPayloadProcessor creates a new payload processor
func NewDefaultPayloadProcessor(detector ContentTypeDetector) *DefaultPayloadProcessor {
	return &DefaultPayloadProcessor{
		contentTypeDetector: detector,
		multipartProcessor:  NewMultipartPayloadProcessor(detector),
	}
}

// Process processes different types of payloads
func (p *DefaultPayloadProcessor) Process(requestID string, data []byte, contentType string, filename string) ([]ProcessedPayload, error) {
	normalizedContentType := p.contentTypeDetector.DetectFromContentType(contentType)

	if strings.HasPrefix(normalizedContentType, "multipart/form-data") {
		return p.multipartProcessor.Process(requestID, data, contentType, filename)
	}

	// Single payload processing
	objectName := p.generateObjectName(requestID, filename, normalizedContentType)

	// Use the most appropriate content type
	finalContentType := normalizedContentType
	if filename != "" {
		fileBasedContentType := p.contentTypeDetector.DetectFromFilename(filename)
		if fileBasedContentType != "application/octet-stream" {
			finalContentType = fileBasedContentType
		}
	}

	return []ProcessedPayload{
		{
			ObjectName:  objectName,
			Data:        data,
			ContentType: finalContentType,
			Filename:    filename,
		},
	}, nil
}

func (p *DefaultPayloadProcessor) generateObjectName(requestID, originalFilename, contentType string) string {
	if originalFilename != "" {
		ext := filepath.Ext(originalFilename)
		base := strings.TrimSuffix(filepath.Base(originalFilename), ext)
		return fmt.Sprintf("%s_%s%s", requestID, base, ext)
	}

	// Generate filename based on content type
	var ext string
	switch {
	case strings.Contains(contentType, "json"):
		ext = ".json"
	case strings.Contains(contentType, "text"):
		ext = ".txt"
	case strings.Contains(contentType, "image"):
		ext = ".img"
	case strings.Contains(contentType, "multipart"):
		ext = ".multipart"
	default:
		ext = ".bin"
	}

	return fmt.Sprintf("%s_payload%s", requestID, ext)
}
