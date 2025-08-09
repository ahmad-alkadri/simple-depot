package main

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDepotEndpoint(t *testing.T) {
	// Start the server in a goroutine
	srv := &http.Server{
		Addr:    ":3003",
		Handler: http.HandlerFunc(depotHandler),
	}

	go func() { _ = srv.ListenAndServe() }()

	serverURL := "http://localhost:3003/depot"

	// Wait for server to be up by polling
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(serverURL)
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

	// Table-driven test cases for method and payload
	type testCase struct {
		name    string
		method  string
		payload []byte
		want    int
	}

	// Use deterministic payloads for repeatability
	smallPayload := []byte("hello world")
	mediumPayload := bytes.Repeat([]byte("a"), 1<<20) // 1MB
	largePayload := bytes.Repeat([]byte("b"), 10<<20) // 10MB

	cases := []testCase{
		{"GET_small", "GET", smallPayload, http.StatusOK},
		{"POST_small", "POST", smallPayload, http.StatusOK},
		{"PUT_small", "PUT", smallPayload, http.StatusOK},
		{"DELETE_small", "DELETE", smallPayload, http.StatusOK},
		{"PATCH_small", "PATCH", smallPayload, http.StatusOK},
		{"OPTIONS_small", "OPTIONS", smallPayload, http.StatusOK},
		{"HEAD_small", "HEAD", smallPayload, http.StatusOK},
		{"POST_medium", "POST", mediumPayload, http.StatusOK},
		{"POST_large", "POST", largePayload, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // safe to run in parallel

			client := &http.Client{Timeout: 60 * time.Second}
			req, err := http.NewRequest(tc.method, serverURL, bytes.NewReader(tc.payload))

			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tc.method != "GET" && tc.method != "HEAD" {
				req.Header.Set("Content-Type", "application/octet-stream")
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Errorf("Expected %d, got %d", tc.want, resp.StatusCode)
			}
		})
	}

	t.Run("POST_multipart_form", func(t *testing.T) {
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
	})
}

// To run: go test -v depot_test.go
