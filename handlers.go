package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type DepotAPI struct {
	Storage StorageService
}

// generateUniqueID creates a unique identifier for each request
func generateUniqueID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to nanoseconds if random fails
		return fmt.Sprintf("%d_%d", timestamp, time.Now().UnixNano())
	}
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%d_%s", timestamp, randomHex)
}

// extractFilenameFromRequest attempts to extract filename from request
func extractFilenameFromRequest(r *http.Request) string {
	// Check Content-Disposition header first
	contentDisposition := r.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		if idx := strings.Index(contentDisposition, "filename="); idx != -1 {
			start := idx + 9 // len("filename=")
			filename := contentDisposition[start:]
			filename = strings.Trim(filename, "\"")
			return filepath.Base(filename)
		}
	}
	return ""
}

// generateObjectName creates a unique object name for storage
func generateObjectName(requestID, originalFilename, contentType string) string {
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

func (api *DepotAPI) DepotHandler(w http.ResponseWriter, r *http.Request) {
	reqTime := time.Now().Format(time.RFC3339)
	requestID := generateUniqueID()

	// Read full body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	originalFilename := extractFilenameFromRequest(r)

	// Process the request
	storage := api.Storage
	go func(body []byte, contentType string, reqTimeStamp string, reqID string) {
		if storage == nil {
			log.Printf("Storage service not initialized")
			return
		}

		if strings.HasPrefix(contentType, "application/json") {
			// JSON payload
			objectName := generateObjectName(reqID, originalFilename, contentType)
			err := storage.SavePayload(objectName, body, "application/json")
			if err != nil {
				log.Printf("Error saving JSON file to storage: %v", err)
				return
			}
			log.Printf("Saved %s as JSON file to storage, reqTime: %s, reqID: %s", objectName, reqTimeStamp, reqID)

		} else if strings.HasPrefix(contentType, "multipart/form-data") {
			// multipart form files
			_, params, err := mime.ParseMediaType(contentType)
			if err != nil {
				log.Printf("Error parsing media type: %v", err)
				return
			}
			boundary := params["boundary"]
			mr := multipart.NewReader(bytes.NewReader(body), boundary)
			fileCount := 0
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("Error reading part: %v", err)
					break
				}
				receivedFileName := part.FileName()
				if receivedFileName == "" {
					continue
				}

				// Read the part data
				partData, err := io.ReadAll(part)
				if err != nil {
					log.Printf("Error reading part data: %v", err)
					continue
				}

				// Use consistent filename with request ID
				uniqueFileName := generateObjectName(reqID, receivedFileName, "")

				// Determine content type based on file extension
				fileContentType := "application/octet-stream"
				ext := strings.ToLower(filepath.Ext(receivedFileName))
				switch ext {
				case ".json":
					fileContentType = "application/json"
				case ".txt":
					fileContentType = "text/plain"
				case ".pdf":
					fileContentType = "application/pdf"
				case ".jpg", ".jpeg":
					fileContentType = "image/jpeg"
				case ".png":
					fileContentType = "image/png"
				case ".gif":
					fileContentType = "image/gif"
				}

				err = storage.SavePayload(uniqueFileName, partData, fileContentType)
				if err != nil {
					log.Printf("Error saving multipart file to storage: %v", err)
					continue
				}
				log.Printf("Saved %s to storage, reqTime: %s, reqID: %s", uniqueFileName, reqTimeStamp, reqID)
				fileCount++
			}
			log.Printf("Saved %d file(s) from multipart body to storage, reqTime: %s, reqID: %s", fileCount, reqTimeStamp, reqID)

		} else {
			// other payloads - use unique object name
			objectName := generateObjectName(reqID, originalFilename, contentType)
			err := storage.SavePayload(objectName, body, contentType)
			if err != nil {
				log.Printf("Error saving binary file to storage: %v", err)
				return
			}
			log.Printf("Saved %s as binary file to storage, reqTime: %s, reqID: %s", objectName, reqTimeStamp, reqID)
		}
	}(bodyBytes, contentType, reqTime, requestID)

	// Prepare response with request ID and metadata
	response := map[string]interface{}{
		"status":     "accepted",
		"request_id": requestID,
		"size":       len(bodyBytes),
		"timestamp":  reqTime,
	}

	if originalFilename != "" {
		response["original_filename"] = originalFilename
	}

	// Log and respond
	log.Printf("[%s] %s request, payload size: %d bytes, request_id: %s", reqTime, r.Method, len(bodyBytes), requestID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ListHandler provides an endpoint to list all stored payloads

func (api *DepotAPI) ListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if api.Storage == nil {
		http.Error(w, "Storage service not initialized", http.StatusInternalServerError)
		return
	}

	objects, err := api.Storage.ListPayloads()
	if err != nil {
		log.Printf("Error listing payloads: %v", err)
		http.Error(w, "Error listing payloads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"count":   len(objects),
		"objects": objects,
	}

	json.NewEncoder(w).Encode(response)
}
