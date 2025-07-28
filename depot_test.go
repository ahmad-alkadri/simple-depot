package main

import (
	"bytes"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDepotEndpoint(t *testing.T) {
	serverURL := "http://localhost:3003/depot"
	types := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

	// Small payload
	smallPayload := []byte("hello world")
	// Medium payload (1MB)
	mediumPayload := make([]byte, 1<<20)
	rand.Read(mediumPayload)
	// Large payload (10MB for test, not 16GB)
	largePayload := make([]byte, 10<<20)
	rand.Read(largePayload)

	payloads := [][]byte{smallPayload, mediumPayload, largePayload}
	payloadNames := []string{"small", "medium", "large"}

	for i, payload := range payloads {
		for _, method := range types {
			t.Run(method+"_"+payloadNames[i], func(t *testing.T) {
				client := &http.Client{Timeout: 60 * time.Second}
				req, err := http.NewRequest(method, serverURL, bytes.NewReader(payload))
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
				if method != "GET" && method != "HEAD" {
					req.Header.Set("Content-Type", "application/octet-stream")
				}
				resp, err := client.Do(req)
				if err != nil {
					t.Fatalf("Request failed: %v", err)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
				}
			})
		}
	}

	// Multipart form test
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.WriteField("field1", strings.Repeat("A", 1024))
	w.Close()
	req, err := http.NewRequest("POST", serverURL, &b)
	if err != nil {
		t.Fatalf("Failed to create multipart request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Multipart request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK for multipart, got %d", resp.StatusCode)
	}
}

// To run: go test -v depot_test.go
