package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"
)

// DefaultPayloadService orchestrates payload operations
type DefaultPayloadService struct {
	storage           StorageService
	processor         PayloadProcessor
	idGenerator       IDGenerator
	responseFormatter ResponseFormatter
	zipService        ZipService
}

// NewDefaultPayloadService creates a new payload service with all dependencies
func NewDefaultPayloadService(
	storage StorageService,
	processor PayloadProcessor,
	idGenerator IDGenerator,
	responseFormatter ResponseFormatter,
	zipService ZipService,
) *DefaultPayloadService {
	return &DefaultPayloadService{
		storage:           storage,
		processor:         processor,
		idGenerator:       idGenerator,
		responseFormatter: responseFormatter,
		zipService:        zipService,
	}
}

// StorePayload processes and stores payload data
func (s *DefaultPayloadService) StorePayload(data []byte, contentType string, filename string) (string, error) {
	requestID := s.idGenerator.Generate()
	reqTime := time.Now().Format(time.RFC3339)

	// Process the payload
	payloads, err := s.processor.Process(requestID, data, contentType, filename)
	if err != nil {
		return "", fmt.Errorf("error processing payload: %v", err)
	}

	// Store payloads asynchronously
	go func(payloads []ProcessedPayload, reqTimeStamp, reqID string) {
		for _, payload := range payloads {
			err := s.storage.SavePayload(payload.ObjectName, payload.Data, payload.ContentType)
			if err != nil {
				log.Printf("Error saving payload to storage: %v", err)
				continue
			}
			log.Printf("Saved %s to storage, reqTime: %s, reqID: %s", payload.ObjectName, reqTimeStamp, reqID)
		}
		log.Printf("Saved %d file(s) to storage, reqTime: %s, reqID: %s", len(payloads), reqTimeStamp, reqID)
	}(payloads, reqTime, requestID)

	return requestID, nil
}

// RetrievePayloads retrieves payloads for a given request ID
func (s *DefaultPayloadService) RetrievePayloads(requestID string, raw bool) (interface{}, error) {
	// List all objects and filter by request_id prefix
	objects, err := s.storage.ListPayloads()
	if err != nil {
		return nil, fmt.Errorf("error listing payloads: %v", err)
	}

	var matched []FileInfo
	for _, obj := range objects {
		if strings.HasPrefix(obj, requestID+"_") || strings.HasPrefix(obj, requestID+"_payload") {
			payload, err := s.storage.GetPayload(obj)
			if err != nil {
				log.Printf("Error getting payload for %s: %v", obj, err)
				continue
			}

			// Determine content type and original filename
			contentType := s.determineContentType(obj)
			originalFilename := s.extractOriginalFilename(obj)

			fileInfo := s.responseFormatter.FormatFileInfo(obj, originalFilename, payload, contentType)
			matched = append(matched, fileInfo)
		}
	}

	if len(matched) == 0 {
		return nil, fmt.Errorf("no payloads found for request_id")
	}

	if raw {
		if len(matched) == 1 {
			// Single file, return raw data
			return s.formatSingleFileResponse(matched[0])
		} else {
			// Multiple files, create zip
			return s.formatZipResponse(matched, requestID)
		}
	}

	// JSON response
	return s.responseFormatter.FormatGetResponse(requestID, matched, len(matched)), nil
}

// ListAllPayloads lists all stored payloads
func (s *DefaultPayloadService) ListAllPayloads() ([]string, error) {
	return s.storage.ListPayloads()
}

func (s *DefaultPayloadService) determineContentType(objectName string) string {
	switch {
	case strings.HasSuffix(objectName, ".json"):
		return "application/json"
	case strings.HasSuffix(objectName, ".txt"):
		return "text/plain"
	case strings.HasSuffix(objectName, ".img"):
		return "application/octet-stream"
	case strings.HasSuffix(objectName, ".multipart"):
		return "multipart/form-data"
	default:
		return "application/octet-stream"
	}
}

func (s *DefaultPayloadService) extractOriginalFilename(objectName string) string {
	parts := strings.Split(objectName, "_")
	if len(parts) > 2 {
		filenameWithExt := strings.Join(parts[2:], "_")
		if strings.HasPrefix(filenameWithExt, "payload") {
			return ""
		}
		return filenameWithExt
	}
	return ""
}

func (s *DefaultPayloadService) formatSingleFileResponse(file FileInfo) (map[string]interface{}, error) {
	filename := file.OriginalFilename
	if filename == "" {
		filename = file.ObjectName
	}

	decoded, err := base64.StdEncoding.DecodeString(file.PayloadBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file: %v", err)
	}

	return map[string]interface{}{
		"filename":     filename,
		"content_type": file.ContentType,
		"data":         decoded,
	}, nil
}

func (s *DefaultPayloadService) formatZipResponse(files []FileInfo, requestID string) (map[string]interface{}, error) {
	zipData, err := s.zipService.CreateZip(files)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip: %v", err)
	}

	return map[string]interface{}{
		"filename":     fmt.Sprintf("payloads_%s.zip", requestID),
		"content_type": "application/zip",
		"data":         zipData,
	}, nil
}
