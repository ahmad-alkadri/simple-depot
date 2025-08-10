package tests

import (
	"fmt"
	"sync"

	"github.com/ahmad-alkadri/simple-depot/internal/handlers"
	"github.com/ahmad-alkadri/simple-depot/internal/services"
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
	return nil, fmt.Errorf("object not found: %s", objectName)
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

func (m *MockStorageService) DeletePayload(objectName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.payloads[objectName]; exists {
		delete(m.payloads, objectName)
		delete(m.contentTypes, objectName)
		return nil
	}
	return fmt.Errorf("object not found: %s", objectName)
}

func (m *MockStorageService) SetSaveError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveError = err
}

func (m *MockStorageService) SetListError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listError = err
}

// createTestHandler creates a handler with all dependencies for testing
func createTestHandler(storage services.StorageService) *handlers.HTTPHandler {
	idGenerator := services.NewDefaultIDGenerator()
	contentTypeDetector := services.NewDefaultContentTypeDetector()
	filenameExtractor := services.NewDefaultFilenameExtractor()
	responseFormatter := services.NewDefaultResponseFormatter()
	zipService := services.NewDefaultZipService()
	payloadProcessor := services.NewDefaultPayloadProcessor(contentTypeDetector)

	payloadService := services.NewDefaultPayloadService(
		storage,
		payloadProcessor,
		idGenerator,
		responseFormatter,
		zipService,
	)

	return handlers.NewHTTPHandler(payloadService, responseFormatter, filenameExtractor)
}
