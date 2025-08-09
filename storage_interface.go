package main

// StorageService interface for storage operations
type StorageService interface {
	SavePayload(objectName string, data []byte, contentType string) error
	GetPayload(objectName string) ([]byte, error)
	ListPayloads() ([]string, error)
}

// Ensure MinioService implements StorageService
var _ StorageService = (*MinioService)(nil)
