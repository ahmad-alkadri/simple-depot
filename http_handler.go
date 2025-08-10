package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// HTTPHandler handles HTTP requests and responses
type HTTPHandler struct {
	payloadService    PayloadService
	responseFormatter ResponseFormatter
	filenameExtractor FilenameExtractor
}

// NewHTTPHandler creates a new HTTP handler with dependencies
func NewHTTPHandler(
	payloadService PayloadService,
	responseFormatter ResponseFormatter,
	filenameExtractor FilenameExtractor,
) *HTTPHandler {
	return &HTTPHandler{
		payloadService:    payloadService,
		responseFormatter: responseFormatter,
		filenameExtractor: filenameExtractor,
	}
}

// DepotHandler handles depot endpoint requests
func (h *HTTPHandler) DepotHandler(w http.ResponseWriter, r *http.Request) {
	reqTime := time.Now().Format(time.RFC3339)

	// Read full body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	originalFilename := h.filenameExtractor.Extract(r.Header.Get("Content-Disposition"))

	// Store the payload
	requestID, err := h.payloadService.StorePayload(bodyBytes, contentType, originalFilename)
	if err != nil {
		log.Printf("Error storing payload: %v", err)
		http.Error(w, "Error storing payload", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := h.responseFormatter.FormatDepotResponse(requestID, len(bodyBytes), reqTime, originalFilename)

	// Log and respond
	log.Printf("[%s] %s request, payload size: %d bytes, request_id: %s", reqTime, r.Method, len(bodyBytes), requestID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetHandler retrieves the payload for a given request_id
func (h *HTTPHandler) GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		http.Error(w, "Missing request_id query parameter", http.StatusBadRequest)
		return
	}

	raw := r.URL.Query().Get("raw") == "true"

	result, err := h.payloadService.RetrievePayloads(requestID, raw)
	if err != nil {
		log.Printf("Error retrieving payloads: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if raw {
		// Handle raw response (single file or zip)
		rawResponse, ok := result.(map[string]interface{})
		if !ok {
			http.Error(w, "Invalid response format", http.StatusInternalServerError)
			return
		}

		filename := rawResponse["filename"].(string)
		contentType := rawResponse["content_type"].(string)
		data := rawResponse["data"].([]byte)

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	// JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// ListHandler provides an endpoint to list all stored payloads
func (h *HTTPHandler) ListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	objects, err := h.payloadService.ListAllPayloads()
	if err != nil {
		log.Printf("Error listing payloads: %v", err)
		http.Error(w, "Error listing payloads", http.StatusInternalServerError)
		return
	}

	response := h.responseFormatter.FormatListResponse(objects, len(objects))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
