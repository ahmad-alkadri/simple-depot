package services

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// MultipartPayloadProcessor handles multipart form data processing
type MultipartPayloadProcessor struct {
	contentTypeDetector ContentTypeDetector
}

// NewMultipartPayloadProcessor creates a new multipart processor
func NewMultipartPayloadProcessor(detector ContentTypeDetector) *MultipartPayloadProcessor {
	return &MultipartPayloadProcessor{
		contentTypeDetector: detector,
	}
}

// Process processes multipart form data into individual payloads
func (p *MultipartPayloadProcessor) Process(requestID string, data []byte, contentType string, filename string) ([]ProcessedPayload, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("error parsing media type: %v", err)
	}

	boundary := params["boundary"]
	mr := multipart.NewReader(bytes.NewReader(data), boundary)

	var payloads []ProcessedPayload

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading part: %v", err)
		}

		receivedFileName := part.FileName()
		if receivedFileName == "" {
			continue
		}

		// Read the part data
		partData, err := io.ReadAll(part)
		if err != nil {
			continue
		}

		// Generate object name
		objectName := p.generateObjectName(requestID, receivedFileName)

		// Detect content type
		fileContentType := p.contentTypeDetector.DetectFromFilename(receivedFileName)

		payloads = append(payloads, ProcessedPayload{
			ObjectName:  objectName,
			Data:        partData,
			ContentType: fileContentType,
			Filename:    receivedFileName,
		})
	}

	return payloads, nil
}

func (p *MultipartPayloadProcessor) generateObjectName(requestID, filename string) string {
	if filename == "" {
		return fmt.Sprintf("%s_payload.bin", requestID)
	}

	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filepath.Base(filename), ext)
	return fmt.Sprintf("%s_%s%s", requestID, base, ext)
}
