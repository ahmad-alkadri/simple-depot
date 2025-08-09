package main

import (
	"bytes"
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

func (api *DepotAPI) DepotHandler(w http.ResponseWriter, r *http.Request) {
	reqTime := time.Now().Format(time.RFC3339)
	method := r.Method

	// Read full body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	size := int64(len(bodyBytes))
	contentType := r.Header.Get("Content-Type")

	// Only start goroutine if body was read successfully
	storage := api.Storage
	go func(body []byte, contentType string, reqTimeStamp string) {
		if storage == nil {
			log.Printf("Storage service not initialized")
			return
		}

		if strings.HasPrefix(contentType, "application/json") {
			// JSON payload
			jsonFileName := fmt.Sprintf("%d.json", time.Now().UnixNano())
			err := storage.SavePayload(jsonFileName, body, "application/json")
			if err != nil {
				log.Printf("Error saving JSON file to storage: %v", err)
				return
			}
			log.Printf("Saved %s as JSON file to storage, reqTime: %s", jsonFileName, reqTimeStamp)

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

				// Generate unique filename to avoid conflicts
				uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), receivedFileName)

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
				fileCount++
			}
			log.Printf("Saved %d file(s) from multipart body to storage, reqTime: %s", fileCount, reqTimeStamp)

		} else {
			// other payloads
			fname := fmt.Sprintf("%d.bin", time.Now().UnixNano())
			err := storage.SavePayload(fname, body, contentType)
			if err != nil {
				log.Printf("Error saving binary file to storage: %v", err)
				return
			}
			log.Printf("Saved %s as binary file to storage, reqTime: %s", fname, reqTimeStamp)
		}
	}(bodyBytes, contentType, reqTime)

	// Log and respond only if body was read successfully
	log.Printf("[%s] %s request, payload size: %d bytes", reqTime, method, size)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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
