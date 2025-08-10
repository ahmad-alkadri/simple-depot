//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// httpGetOrLaunchServer checks if the server is running, launches it if not, and waits for it to be ready
func httpGetOrLaunchServer(baseURL string, t *testing.T) error {
	resp, err := http.Get(baseURL + "/list")
	if err == nil {
		resp.Body.Close()
		return nil
	}
	// Launch the server in a goroutine
	config := LoadConfig()
	storageService, err := NewMinioService(config)
	if err != nil {
		t.Fatalf("Failed to initialize MinIO service: %v", err)
	}

	// Create all service dependencies (following dependency injection)
	idGenerator := NewDefaultIDGenerator()
	contentTypeDetector := NewDefaultContentTypeDetector()
	filenameExtractor := NewDefaultFilenameExtractor()
	responseFormatter := NewDefaultResponseFormatter()
	zipService := NewDefaultZipService()
	payloadProcessor := NewDefaultPayloadProcessor(contentTypeDetector)

	// Create payload service with all dependencies
	payloadService := NewDefaultPayloadService(
		storageService,
		payloadProcessor,
		idGenerator,
		responseFormatter,
		zipService,
	)

	// Create HTTP handler with dependencies
	httpHandler := NewHTTPHandler(payloadService, responseFormatter, filenameExtractor)

	mux := http.NewServeMux()
	mux.HandleFunc("/depot", httpHandler.DepotHandler)
	mux.HandleFunc("/list", httpHandler.ListHandler)
	mux.HandleFunc("/get", httpHandler.GetHandler)
	srv := &http.Server{
		Addr:    ":" + config.ServerPort,
		Handler: mux,
	}
	go func() {
		_ = srv.ListenAndServe()
	}()

	// Wait for server to be up by polling
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(baseURL + "/list")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Server did not start within timeout: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	})
	return nil
}

// TestServerIntegration tests the full server with live MinIO
// Prerequisites: Server must be running on localhost:3003, MinIO must be running
func TestServerIntegration(t *testing.T) {
	// Check if server is running, if not, launch it
	baseURL := "http://localhost:3003"
	err := httpGetOrLaunchServer(baseURL, t)

	// Check if MinIO is accessible
	minioEndpoint := LoadConfig().MinioEndpoint
	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		t.Skipf("Skipping server integration test: MinIO not accessible: %v", err)
	}

	bucket := LoadConfig().MinioBucket
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil || !exists {
		t.Skipf("Skipping server integration test: MinIO bucket %s not accessible", bucket)
	}

	t.Run("JSONPayload_UniqueFilenames", func(t *testing.T) {
		testJSONPayloadWithUniqueFilenames(t, baseURL, minioClient, bucket)
	})

	t.Run("BinaryPayload_UniqueFilenames", func(t *testing.T) {
		testBinaryPayloadWithUniqueFilenames(t, baseURL, minioClient, bucket)
	})

	t.Run("MultipartPayload_UniqueFilenames", func(t *testing.T) {
		testMultipartPayloadWithUniqueFilenames(t, baseURL, minioClient, bucket)
	})

	t.Run("SameFilename_DifferentContent", func(t *testing.T) {
		testSameFilenameWithDifferentContent(t, baseURL, minioClient, bucket)
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		testConcurrentRequests(t, baseURL, minioClient, bucket)
	})

	t.Run("ListEndpoint_VerifyFiles", func(t *testing.T) {
		testListEndpointVerifyFiles(t, baseURL)
	})
}

func testJSONPayloadWithUniqueFilenames(t *testing.T, baseURL string, minioClient *minio.Client, bucket string) {
	jsonPayloads := []string{
		`{"test": "json1", "timestamp": "2024-01-01T10:00:00Z"}`,
		`{"test": "json2", "data": [1, 2, 3, 4, 5]}`,
		`{"test": "json3", "nested": {"key": "value", "number": 42}}`,
	}

	var requestIDs []string

	for i, payload := range jsonPayloads {
		t.Logf("Sending JSON payload %d", i+1)

		resp, err := http.Post(baseURL+"/depot", "application/json", strings.NewReader(payload))
		if err != nil {
			t.Fatalf("Failed to send JSON payload %d: %v", i+1, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d for payload %d", resp.StatusCode, i+1)
		}

		// Parse response to get request ID
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response for payload %d: %v", i+1, err)
		}

		requestID, ok := response["request_id"].(string)
		if !ok {
			t.Fatalf("Missing request_id in response for payload %d", i+1)
		}
		requestIDs = append(requestIDs, requestID)
		t.Logf("Received request_id: %s", requestID)
	}

	// Wait for async processing
	time.Sleep(2 * time.Second)

	// Verify files exist in MinIO with unique names
	ctx := context.Background()
	for i, requestID := range requestIDs {
		expectedFilename := fmt.Sprintf("%s_payload.json", requestID)

		// Check if object exists
		_, err := minioClient.StatObject(ctx, bucket, expectedFilename, minio.StatObjectOptions{})
		if err != nil {
			t.Errorf("File %s not found in MinIO for request %d: %v", expectedFilename, i+1, err)
			continue
		}

		// Verify content
		obj, err := minioClient.GetObject(ctx, bucket, expectedFilename, minio.GetObjectOptions{})
		if err != nil {
			t.Errorf("Failed to get object %s: %v", expectedFilename, err)
			continue
		}

		content, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			t.Errorf("Failed to read object %s: %v", expectedFilename, err)
			continue
		}

		if string(content) != jsonPayloads[i] {
			t.Errorf("Content mismatch for %s. Expected: %s, Got: %s", expectedFilename, jsonPayloads[i], string(content))
		}
	}
}

func testBinaryPayloadWithUniqueFilenames(t *testing.T, baseURL string, minioClient *minio.Client, bucket string) {
	binaryPayloads := [][]byte{
		{0x00, 0x01, 0x02, 0x03, 0xFF},
		{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		{0x12, 0x34, 0x56, 0x78, 0x90, 0xAB, 0xCD, 0xEF},
	}

	var requestIDs []string

	for i, payload := range binaryPayloads {
		t.Logf("Sending binary payload %d", i+1)

		req, err := http.NewRequest("POST", baseURL+"/depot", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("Failed to create request for payload %d: %v", i+1, err)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"test%d.bin\"", i+1))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to send binary payload %d: %v", i+1, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d for payload %d", resp.StatusCode, i+1)
		}

		// Parse response to get request ID
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response for payload %d: %v", i+1, err)
		}

		requestID, ok := response["request_id"].(string)
		if !ok {
			t.Fatalf("Missing request_id in response for payload %d", i+1)
		}
		requestIDs = append(requestIDs, requestID)
		t.Logf("Received request_id: %s", requestID)
	}

	// Wait for async processing
	time.Sleep(2 * time.Second)

	// Verify files exist in MinIO
	ctx := context.Background()
	for i, requestID := range requestIDs {
		expectedFilename := fmt.Sprintf("%s_test%d.bin", requestID, i+1)

		// Check if object exists
		_, err := minioClient.StatObject(ctx, bucket, expectedFilename, minio.StatObjectOptions{})
		if err != nil {
			t.Errorf("File %s not found in MinIO for request %d: %v", expectedFilename, i+1, err)
			continue
		}

		// Verify content
		obj, err := minioClient.GetObject(ctx, bucket, expectedFilename, minio.GetObjectOptions{})
		if err != nil {
			t.Errorf("Failed to get object %s: %v", expectedFilename, err)
			continue
		}

		content, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			t.Errorf("Failed to read object %s: %v", expectedFilename, err)
			continue
		}

		if !bytes.Equal(content, binaryPayloads[i]) {
			t.Errorf("Content mismatch for %s. Expected: %v, Got: %v", expectedFilename, binaryPayloads[i], content)
		}
	}
}

func testMultipartPayloadWithUniqueFilenames(t *testing.T, baseURL string, minioClient *minio.Client, bucket string) {
	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add multiple files with same names but different content
	files := []struct {
		filename string
		content  string
	}{
		{"test.txt", "This is test file 1"},
		{"data.json", `{"id": 1, "name": "first"}`},
		{"binary.bin", "Binary data 1"},
	}

	for _, file := range files {
		part, err := writer.CreateFormFile("files", file.filename)
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write([]byte(file.content))
	}
	writer.Close()

	// Send multipart request
	req, err := http.NewRequest("POST", baseURL+"/depot", &buf)
	if err != nil {
		t.Fatalf("Failed to create multipart request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send multipart payload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response to get request ID
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	requestID, ok := response["request_id"].(string)
	if !ok {
		t.Fatalf("Missing request_id in response")
	}
	t.Logf("Received request_id: %s", requestID)

	// Wait for async processing
	time.Sleep(2 * time.Second)

	// Verify files exist in MinIO with request ID prefix
	ctx := context.Background()
	for _, file := range files {
		expectedFilename := fmt.Sprintf("%s_%s", requestID, file.filename)

		// Check if object exists
		_, err := minioClient.StatObject(ctx, bucket, expectedFilename, minio.StatObjectOptions{})
		if err != nil {
			t.Errorf("File %s not found in MinIO: %v", expectedFilename, err)
			continue
		}

		// Verify content
		obj, err := minioClient.GetObject(ctx, bucket, expectedFilename, minio.GetObjectOptions{})
		if err != nil {
			t.Errorf("Failed to get object %s: %v", expectedFilename, err)
			continue
		}

		content, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			t.Errorf("Failed to read object %s: %v", expectedFilename, err)
			continue
		}

		if string(content) != file.content {
			t.Errorf("Content mismatch for %s. Expected: %s, Got: %s", expectedFilename, file.content, string(content))
		}
	}
}

func testSameFilenameWithDifferentContent(t *testing.T, baseURL string, minioClient *minio.Client, bucket string) {
	filename := "same_file.txt"
	contents := []string{
		"This is version 1 of the file",
		"This is version 2 of the file",
		"This is version 3 of the file",
	}

	var requestIDs []string

	// Send multiple requests with same filename but different content
	for i, content := range contents {
		t.Logf("Sending file %s (version %d)", filename, i+1)

		req, err := http.NewRequest("POST", baseURL+"/depot", strings.NewReader(content))
		if err != nil {
			t.Fatalf("Failed to create request %d: %v", i+1, err)
		}
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request %d: %v", i+1, err)
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response %d: %v", i+1, err)
		}

		requestID, ok := response["request_id"].(string)
		if !ok {
			t.Fatalf("Missing request_id in response %d", i+1)
		}
		requestIDs = append(requestIDs, requestID)
	}

	// Wait for async processing
	time.Sleep(2 * time.Second)

	// Verify all three files exist with different names and correct content
	ctx := context.Background()
	for i, requestID := range requestIDs {
		expectedFilename := fmt.Sprintf("%s_same_file.txt", requestID)

		obj, err := minioClient.GetObject(ctx, bucket, expectedFilename, minio.GetObjectOptions{})
		if err != nil {
			t.Errorf("Failed to get object %s: %v", expectedFilename, err)
			continue
		}

		content, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			t.Errorf("Failed to read object %s: %v", expectedFilename, err)
			continue
		}

		if string(content) != contents[i] {
			t.Errorf("Content mismatch for %s. Expected: %s, Got: %s", expectedFilename, contents[i], string(content))
		}
	}
}

func testConcurrentRequests(t *testing.T, baseURL string, minioClient *minio.Client, bucket string) {
	numRequests := 10
	done := make(chan string, numRequests)

	// Send concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			payload := fmt.Sprintf(`{"concurrent_test": true, "request_number": %d, "timestamp": "%s"}`,
				id, time.Now().Format(time.RFC3339))

			resp, err := http.Post(baseURL+"/depot", "application/json", strings.NewReader(payload))
			if err != nil {
				t.Errorf("Failed to send concurrent request %d: %v", id, err)
				done <- ""
				return
			}
			defer resp.Body.Close()

			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Errorf("Failed to decode response %d: %v", id, err)
				done <- ""
				return
			}

			requestID, ok := response["request_id"].(string)
			if !ok {
				t.Errorf("Missing request_id in response %d", id)
				done <- ""
				return
			}

			done <- requestID
		}(i)
	}

	// Collect all request IDs
	var requestIDs []string
	for i := 0; i < numRequests; i++ {
		id := <-done
		if id != "" {
			requestIDs = append(requestIDs, id)
		}
	}

	if len(requestIDs) != numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, len(requestIDs))
	}

	// Wait for async processing
	time.Sleep(3 * time.Second)

	// Verify all files were saved with unique names
	ctx := context.Background()
	for i, requestID := range requestIDs {
		expectedFilename := fmt.Sprintf("%s_payload.json", requestID)

		_, err := minioClient.StatObject(ctx, bucket, expectedFilename, minio.StatObjectOptions{})
		if err != nil {
			t.Errorf("Concurrent request %d file %s not found in MinIO: %v", i, expectedFilename, err)
		}
	}

	// Verify all request IDs are unique
	uniqueIDs := make(map[string]bool)
	for _, id := range requestIDs {
		if uniqueIDs[id] {
			t.Errorf("Duplicate request ID found: %s", id)
		}
		uniqueIDs[id] = true
	}
}

func testListEndpointVerifyFiles(t *testing.T, baseURL string) {
	// Send a test file first
	testPayload := `{"list_test": true, "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`
	resp, err := http.Post(baseURL+"/depot", "application/json", strings.NewReader(testPayload))
	if err != nil {
		t.Fatalf("Failed to send test payload: %v", err)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	requestID, ok := response["request_id"].(string)
	if !ok {
		t.Fatalf("Missing request_id in response")
	}

	// Wait for async processing
	time.Sleep(2 * time.Second)

	// Test list endpoint
	listResp, err := http.Get(baseURL + "/list")
	if err != nil {
		t.Fatalf("Failed to get list: %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for list endpoint, got %d", listResp.StatusCode)
	}

	var listResponse map[string]interface{}
	if err := json.NewDecoder(listResp.Body).Decode(&listResponse); err != nil {
		t.Fatalf("Failed to decode list response: %v", err)
	}

	objects, ok := listResponse["objects"].([]interface{})
	if !ok {
		t.Fatalf("Missing or invalid objects in list response")
	}

	// Verify our test file is in the list
	expectedFilename := fmt.Sprintf("%s_payload.json", requestID)
	found := false
	for _, obj := range objects {
		if obj.(string) == expectedFilename {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Test file %s not found in list endpoint", expectedFilename)
	}
}
