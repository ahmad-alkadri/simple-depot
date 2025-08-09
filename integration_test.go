//go:build integration
// +build integration

package main

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// Integration tests that require a real MinIO instance
// Run with: go test -tags=integration ./...

func TestMinioService_Integration(t *testing.T) {
	// Skip if no MinIO endpoint configured
	if os.Getenv("MINIO_ENDPOINT") == "" {
		t.Skip("Skipping integration test: MINIO_ENDPOINT not set")
	}

	// Load configuration from environment
	config := LoadConfig()

	// Create MinIO service
	service, err := NewMinioService(config)
	if err != nil {
		t.Fatalf("Failed to create MinIO service: %v", err)
	}

	t.Run("SaveAndGetPayload_JSON", func(t *testing.T) {
		objectName := "test_json_" + time.Now().Format("20060102_150405") + ".json"
		testData := []byte(`{"test": "integration", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`)

		// Save payload
		err := service.SavePayload(objectName, testData, "application/json")
		if err != nil {
			t.Fatalf("Failed to save payload: %v", err)
		}

		// Get payload back
		retrievedData, err := service.GetPayload(objectName)
		if err != nil {
			t.Fatalf("Failed to retrieve payload: %v", err)
		}

		// Verify data matches
		if !bytes.Equal(testData, retrievedData) {
			t.Errorf("Data mismatch. Expected %s, got %s", string(testData), string(retrievedData))
		}
	})

	t.Run("SaveAndGetPayload_Binary", func(t *testing.T) {
		objectName := "test_binary_" + time.Now().Format("20060102_150405") + ".bin"
		testData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xAA, 0xBB}

		// Save payload
		err := service.SavePayload(objectName, testData, "application/octet-stream")
		if err != nil {
			t.Fatalf("Failed to save payload: %v", err)
		}

		// Get payload back
		retrievedData, err := service.GetPayload(objectName)
		if err != nil {
			t.Fatalf("Failed to retrieve payload: %v", err)
		}

		// Verify data matches
		if !bytes.Equal(testData, retrievedData) {
			t.Errorf("Data mismatch. Expected %v, got %v", testData, retrievedData)
		}
	})

	t.Run("ListPayloads", func(t *testing.T) {
		// Create a few test objects
		timestamp := time.Now().Format("20060102_150405")
		testObjects := []string{
			"list_test_1_" + timestamp + ".json",
			"list_test_2_" + timestamp + ".txt",
			"list_test_3_" + timestamp + ".bin",
		}

		// Save test objects
		for _, objName := range testObjects {
			testData := []byte("test data for " + objName)
			err := service.SavePayload(objName, testData, "text/plain")
			if err != nil {
				t.Fatalf("Failed to save test object %s: %v", objName, err)
			}
		}

		// List all payloads
		objects, err := service.ListPayloads()
		if err != nil {
			t.Fatalf("Failed to list payloads: %v", err)
		}

		// Verify our test objects are in the list
		for _, testObj := range testObjects {
			found := false
			for _, obj := range objects {
				if obj == testObj {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Test object %s not found in list", testObj)
			}
		}
	})

	t.Run("SavePayload_LargeFile", func(t *testing.T) {
		objectName := "large_file_" + time.Now().Format("20060102_150405") + ".bin"

		// Create a 1MB test file
		testData := make([]byte, 1024*1024)
		for i := range testData {
			testData[i] = byte(i % 256)
		}

		// Save payload
		err := service.SavePayload(objectName, testData, "application/octet-stream")
		if err != nil {
			t.Fatalf("Failed to save large payload: %v", err)
		}

		// Get payload back
		retrievedData, err := service.GetPayload(objectName)
		if err != nil {
			t.Fatalf("Failed to retrieve large payload: %v", err)
		}

		// Verify size matches
		if len(retrievedData) != len(testData) {
			t.Errorf("Size mismatch. Expected %d bytes, got %d bytes", len(testData), len(retrievedData))
		}

		// Verify data matches (check first and last few bytes for performance)
		if !bytes.Equal(testData[:100], retrievedData[:100]) {
			t.Error("First 100 bytes don't match")
		}
		if !bytes.Equal(testData[len(testData)-100:], retrievedData[len(retrievedData)-100:]) {
			t.Error("Last 100 bytes don't match")
		}
	})
}

func TestMinioService_Integration_ErrorCases(t *testing.T) {
	// Skip if no MinIO endpoint configured
	if os.Getenv("MINIO_ENDPOINT") == "" {
		t.Skip("Skipping integration test: MINIO_ENDPOINT not set")
	}

	t.Run("GetNonExistentObject", func(t *testing.T) {
		config := LoadConfig()
		service, err := NewMinioService(config)
		if err != nil {
			t.Fatalf("Failed to create MinIO service: %v", err)
		}

		// Try to get a non-existent object
		_, err = service.GetPayload("non_existent_file_" + time.Now().Format("20060102_150405"))
		if err == nil {
			t.Error("Expected error when getting non-existent object, but got nil")
		}
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		config := LoadConfig()
		config.MinioAccessKey = "invalid_key"
		config.MinioSecretKey = "invalid_secret"

		// This should fail during bucket operations, not during service creation
		service, err := NewMinioService(config)
		if err == nil {
			// Try to save something, which should fail
			err = service.SavePayload("test.txt", []byte("test"), "text/plain")
			if err == nil {
				t.Error("Expected error with invalid credentials, but operation succeeded")
			}
		}
	})
}
