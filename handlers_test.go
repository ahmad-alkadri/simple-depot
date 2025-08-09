package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockStorageService implements a mock version of StorageService for testing
type MockStorageService struct {
	payloads     map[string][]byte
	contentTypes map[string]string
	saveError    error
	listError    error
	mu           sync.Mutex
}

func NewMockStorageService() *MockStorageService {
	return &MockStorageService{
		payloads:     make(map[string][]byte),
		contentTypes: make(map[string]string),
	}
}

func (m *MockStorageService) SavePayload(objectName string, data []byte, contentType string) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payloads[objectName] = data
	m.contentTypes[objectName] = contentType
	return nil
}

func (m *MockStorageService) GetPayload(objectName string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if data, exists := m.payloads[objectName]; exists {
		return data, nil
	}
	return nil, nil
}

func (m *MockStorageService) ListPayloads() ([]string, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var objects []string
	for key := range m.payloads {
		objects = append(objects, key)
	}
	return objects, nil
}

func TestDepotHandler_JSONPayload(t *testing.T) {
	mockService := NewMockStorageService()
	api := &DepotAPI{Storage: mockService}

	jsonPayload := `{"test": "data", "timestamp": "2025-08-09T10:00:00Z"}`
	req := httptest.NewRequest("POST", "/depot", strings.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	api.DepotHandler(w, req)

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
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	// Verify response fields
	if status, ok := response["status"].(string); !ok || status != "accepted" {
		t.Errorf("Expected status 'accepted', got %v", response["status"])
	}

	if _, ok := response["request_id"].(string); !ok {
		t.Errorf("Expected request_id in response, got %v", response["request_id"])
	}

	if size, ok := response["size"].(float64); !ok || int(size) != len(jsonPayload) {
		t.Errorf("Expected size %d, got %v", len(jsonPayload), response["size"])
	}

	// Allow some time for goroutine to complete
	time.Sleep(10 * time.Millisecond)

	// Verify that payload was saved
	if len(mockService.payloads) != 1 {
		t.Errorf("Expected 1 payload saved, got %d", len(mockService.payloads))
	}

	// Find the JSON file (filename will contain timestamp)
	var jsonFile string
	for key := range mockService.payloads {
		if strings.HasSuffix(key, ".json") {
			jsonFile = key
			break
		}
	}

	if jsonFile == "" {
		t.Error("No JSON file found in saved payloads")
		return
	}

	savedData := mockService.payloads[jsonFile]
	if string(savedData) != jsonPayload {
		t.Errorf("Expected saved data %s, got %s", jsonPayload, string(savedData))
	}

	fileContentType := mockService.contentTypes[jsonFile]
	if fileContentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got %s", fileContentType)
	}
}

func TestDepotHandler_BinaryPayload(t *testing.T) {
	mockService := NewMockStorageService()
	api := &DepotAPI{Storage: mockService}

	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF}
	req := httptest.NewRequest("POST", "/depot", bytes.NewReader(binaryData))
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()

	// Execute
	api.DepotHandler(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	// Allow time for goroutine
	time.Sleep(10 * time.Millisecond)

	// Verify payload was saved
	if len(mockService.payloads) != 1 {
		t.Errorf("Expected 1 payload saved, got %d", len(mockService.payloads))
	}

	// Find the binary file
	var binFile string
	for key := range mockService.payloads {
		if strings.HasSuffix(key, ".bin") {
			binFile = key
			break
		}
	}

	if binFile == "" {
		t.Error("No binary file found in saved payloads")
		return
	}

	savedData := mockService.payloads[binFile]
	if !bytes.Equal(savedData, binaryData) {
		t.Errorf("Expected saved data %v, got %v", binaryData, savedData)
	}
}

func TestDepotHandler_MultipartFormData(t *testing.T) {
	mockService := NewMockStorageService()
	api := &DepotAPI{Storage: mockService}

	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add a file
	fileContent := "test file content"
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte(fileContent))
	writer.Close()

	req := httptest.NewRequest("POST", "/depot", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Execute
	api.DepotHandler(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	// Allow time for goroutine
	time.Sleep(10 * time.Millisecond)

	// Verify file was saved
	if len(mockService.payloads) != 1 {
		t.Errorf("Expected 1 payload saved, got %d", len(mockService.payloads))
	}

	// Find the uploaded file (will have timestamp prefix)
	var uploadedFile string
	for key := range mockService.payloads {
		if strings.Contains(key, "test.txt") {
			uploadedFile = key
			break
		}
	}

	if uploadedFile == "" {
		t.Error("No uploaded file found in saved payloads")
		return
	}

	savedData := mockService.payloads[uploadedFile]
	if string(savedData) != fileContent {
		t.Errorf("Expected saved data %s, got %s", fileContent, string(savedData))
	}
}

func TestDepotHandler_NoStorageService(t *testing.T) {
	api := &DepotAPI{Storage: nil}
	req := httptest.NewRequest("POST", "/depot", strings.NewReader("test"))
	w := httptest.NewRecorder()

	// Execute
	api.DepotHandler(w, req)

	// Should still return OK even if storage service is not available
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}
}

func TestListHandler_Success(t *testing.T) {
	mockService := NewMockStorageService()
	mockService.payloads["file1.json"] = []byte(`{"test": 1}`)
	mockService.payloads["file2.bin"] = []byte{0x01, 0x02}
	api := &DepotAPI{Storage: mockService}

	req := httptest.NewRequest("GET", "/list", nil)
	w := httptest.NewRecorder()

	// Execute
	api.ListHandler(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got %s", contentType)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify count
	count, ok := response["count"].(float64) // JSON numbers are float64
	if !ok || int(count) != 2 {
		t.Errorf("Expected count 2, got %v", response["count"])
	}

	// Verify objects list
	objects, ok := response["objects"].([]interface{})
	if !ok || len(objects) != 2 {
		t.Errorf("Expected 2 objects in list, got %v", response["objects"])
	}
}

func TestListHandler_MethodNotAllowed(t *testing.T) {
	api := &DepotAPI{Storage: NewMockStorageService()}
	req := httptest.NewRequest("POST", "/list", nil)
	w := httptest.NewRecorder()

	// Execute
	api.ListHandler(w, req)

	// Verify error response
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", w.Code)
	}
}

func TestListHandler_NoStorageService(t *testing.T) {
	api := &DepotAPI{Storage: nil}
	req := httptest.NewRequest("GET", "/list", nil)
	w := httptest.NewRecorder()

	// Execute
	api.ListHandler(w, req)

	// Verify error response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status InternalServerError, got %d", w.Code)
	}
}

func TestListHandler_StorageError(t *testing.T) {
	mockService := NewMockStorageService()
	mockService.listError = io.EOF // Simulate an error
	api := &DepotAPI{Storage: mockService}

	req := httptest.NewRequest("GET", "/list", nil)
	w := httptest.NewRecorder()

	// Execute
	api.ListHandler(w, req)

	// Verify error response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status InternalServerError, got %d", w.Code)
	}
}

// Benchmark tests for performance
func BenchmarkDepotHandler_JSONPayload(b *testing.B) {
	mockService := NewMockStorageService()
	api := &DepotAPI{Storage: mockService}

	jsonPayload := `{"test": "data", "timestamp": "2025-08-09T10:00:00Z"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/depot", strings.NewReader(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		api.DepotHandler(w, req)
	}
}

func BenchmarkListHandler(b *testing.B) {
	mockService := NewMockStorageService()
	// Add some test data
	for i := 0; i < 100; i++ {
		mockService.payloads[fmt.Sprintf("file%d.json", i)] = []byte(`{"test": "data"}`)
	}
	api := &DepotAPI{Storage: mockService}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/list", nil)
		w := httptest.NewRecorder()

		api.ListHandler(w, req)
	}
}
