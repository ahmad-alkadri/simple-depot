package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/base64"
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

// GetHandler retrieves the payload for a given request_id
func (api *DepotAPI) GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		http.Error(w, "Missing request_id query parameter", http.StatusBadRequest)
		return
	}

	// List all objects and filter by request_id prefix
	objects, err := api.Storage.ListPayloads()
	if err != nil {
		log.Printf("Error listing payloads: %v", err)
		http.Error(w, "Error listing payloads", http.StatusInternalServerError)
		return
	}

	var matched []map[string]any
	for _, obj := range objects {
		if strings.HasPrefix(obj, requestID+"_") || strings.HasPrefix(obj, requestID+"_payload") {
			payload, err := api.Storage.GetPayload(obj)
			if err != nil {
				log.Printf("Error getting payload for %s: %v", obj, err)
				continue
			}
			// Determine content type
			var contentType string
			switch {
			case strings.HasSuffix(obj, ".json"):
				contentType = "application/json"
			case strings.HasSuffix(obj, ".txt"):
				contentType = "text/plain"
			case strings.HasSuffix(obj, ".img"):
				contentType = "application/octet-stream"
			case strings.HasSuffix(obj, ".multipart"):
				contentType = "multipart/form-data"
			default:
				contentType = "application/octet-stream"
			}
			// Extract original filename from object name
			originalFilename := ""
			parts := strings.Split(obj, "_")
			if len(parts) > 2 {
				filenameWithExt := strings.Join(parts[2:], "_")
				if strings.HasPrefix(filenameWithExt, "payload") {
					originalFilename = ""
				} else {
					originalFilename = filenameWithExt
				}
			}
			matched = append(matched, map[string]any{
				"object_name":       obj,
				"original_filename": originalFilename,
				"size":              len(payload),
				"content_type":      contentType,
				"payload_base64":    encodeToBase64(payload),
			})
		}
	}
	if len(matched) == 0 {
		http.Error(w, "No payloads found for request_id", http.StatusNotFound)
		return
	}

	raw := r.URL.Query().Get("raw")
	if raw == "true" {
		if len(matched) == 1 {
			// Single file, serve directly
			file := matched[0]
			filename := file["original_filename"].(string)
			if filename == "" {
				filename = file["object_name"].(string)
			}
			w.Header().Set("Content-Type", file["content_type"].(string))
			w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
			w.WriteHeader(http.StatusOK)
			// Serve the original bytes, not base64
			payload := file["payload_base64"].(string)
			decoded, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				http.Error(w, "Failed to decode file", http.StatusInternalServerError)
				return
			}
			w.Write(decoded)
			return
		} else {
			// Multiple files, zip them
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", "attachment; filename=\"payloads_"+requestID+".zip\"")
			w.WriteHeader(http.StatusOK)
			zipWriter := NewZipWriter(w)
			for _, file := range matched {
				filename := file["original_filename"].(string)
				if filename == "" {
					filename = file["object_name"].(string)
				}
				payload := file["payload_base64"].(string)
				decoded, err := base64.StdEncoding.DecodeString(payload)
				if err != nil {
					continue
				}
				zipWriter.AddFile(filename, decoded)
			}
			zipWriter.Close()
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"request_id": requestID,
		"files":      matched,
		"count":      len(matched),
	})
}

func encodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
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

type ZipWriter struct {
	zw *zip.Writer
	w  http.ResponseWriter
}

// NewZipWriter wraps archive/zip for streaming zip creation
func NewZipWriter(w http.ResponseWriter) *ZipWriter {
	return &ZipWriter{zw: zip.NewWriter(w), w: w}
}

func (z *ZipWriter) AddFile(filename string, data []byte) error {
	f, err := z.zw.Create(filename)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func (z *ZipWriter) Close() error {
	return z.zw.Close()
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
	response := map[string]any{
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
	response := map[string]any{
		"count":   len(objects),
		"objects": objects,
	}

	json.NewEncoder(w).Encode(response)
}
