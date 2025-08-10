package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDepotHandler_JSONPayload(t *testing.T) {
	mockService := NewMockStorageService()
	handler := createTestHandler(mockService)

	jsonPayload := `{"test": "data", "timestamp": "2025-08-09T10:00:00Z"}`
	req := httptest.NewRequest("POST", "/depot", strings.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.DepotHandler(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	// Verify response is JSON
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got %s", contentType)
	}

	// Parse JSON response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify response fields
	if response["status"] != "accepted" {
		t.Errorf("Expected status 'accepted', got %v", response["status"])
	}

	if response["request_id"] == nil {
		t.Error("Expected request_id in response")
	}

	expectedSize := float64(len(jsonPayload))
	if response["size"] != expectedSize {
		t.Errorf("Expected size %f, got %v", expectedSize, response["size"])
	}

	// Verify timestamp is present and valid
	if response["timestamp"] == nil {
		t.Error("Expected timestamp in response")
	}

	// Wait a bit for async storage
	time.Sleep(100 * time.Millisecond)

	// Verify data was stored
	if len(mockService.payloads) == 0 {
		t.Error("Expected payload to be stored")
	}
}

func TestDepotHandler_BinaryPayload(t *testing.T) {
	mockService := NewMockStorageService()
	handler := createTestHandler(mockService)

	binaryData := []byte("hello")
	req := httptest.NewRequest("POST", "/depot", bytes.NewReader(binaryData))
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()

	// Execute
	handler.DepotHandler(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "accepted" {
		t.Errorf("Expected status 'accepted', got %v", response["status"])
	}

	// Wait for async storage
	time.Sleep(100 * time.Millisecond)

	// Verify data was stored
	if len(mockService.payloads) == 0 {
		t.Error("Expected payload to be stored")
	}
}

func TestDepotHandler_MultipartFormData(t *testing.T) {
	mockService := NewMockStorageService()
	handler := createTestHandler(mockService)

	// Create multipart form data
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte("test file content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/depot", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Execute
	handler.DepotHandler(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "accepted" {
		t.Errorf("Expected status 'accepted', got %v", response["status"])
	}

	// Wait for async storage
	time.Sleep(100 * time.Millisecond)

	// Verify data was stored
	if len(mockService.payloads) == 0 {
		t.Error("Expected payload to be stored")
	}
}

func TestListHandler_Success(t *testing.T) {
	mockService := NewMockStorageService()
	mockService.payloads["test1"] = []byte("data1")
	mockService.payloads["test2"] = []byte("data2")

	handler := createTestHandler(mockService)

	req := httptest.NewRequest("GET", "/list", nil)
	w := httptest.NewRecorder()

	handler.ListHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["count"] != float64(2) {
		t.Errorf("Expected count 2, got %v", response["count"])
	}
}

func TestListHandler_MethodNotAllowed(t *testing.T) {
	mockService := NewMockStorageService()
	handler := createTestHandler(mockService)

	req := httptest.NewRequest("POST", "/list", nil)
	w := httptest.NewRecorder()

	handler.ListHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", w.Code)
	}
}

func TestGetHandler_Success(t *testing.T) {
	mockService := NewMockStorageService()
	// Add test data with proper naming pattern
	testData := []byte("test data")
	mockService.payloads["12345_test.txt"] = testData

	handler := createTestHandler(mockService)

	req := httptest.NewRequest("GET", "/get?request_id=12345", nil)
	w := httptest.NewRecorder()

	handler.GetHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}
}

func TestGetHandler_MissingRequestID(t *testing.T) {
	mockService := NewMockStorageService()
	handler := createTestHandler(mockService)

	req := httptest.NewRequest("GET", "/get", nil)
	w := httptest.NewRecorder()

	handler.GetHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestGetHandler_NotFound(t *testing.T) {
	mockService := NewMockStorageService()
	handler := createTestHandler(mockService)

	req := httptest.NewRequest("GET", "/get?request_id=nonexistent", nil)
	w := httptest.NewRecorder()

	handler.GetHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", w.Code)
	}
}

// Benchmarks
func BenchmarkDepotHandler_JSONPayload(b *testing.B) {
	mockService := NewMockStorageService()
	handler := createTestHandler(mockService)

	jsonPayload := `{"test": "data", "timestamp": "2025-08-09T10:00:00Z"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/depot", strings.NewReader(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.DepotHandler(w, req)
	}
}

func BenchmarkDepotHandler_BinaryPayload(b *testing.B) {
	mockService := NewMockStorageService()
	handler := createTestHandler(mockService)

	binaryData := make([]byte, 1024) // 1KB binary data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/depot", bytes.NewReader(binaryData))
		req.Header.Set("Content-Type", "application/octet-stream")
		w := httptest.NewRecorder()

		handler.DepotHandler(w, req)
	}
}
