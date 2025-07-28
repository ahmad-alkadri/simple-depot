package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func depotHandler(w http.ResponseWriter, r *http.Request) {
	reqTime := time.Now().Format(time.RFC3339)
	method := r.Method

	// Read full body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
	}
	r.Body.Close()
	size := int64(len(bodyBytes))
	contentType := r.Header.Get("Content-Type")

	// Save body or parts asynchronously
	go func(body []byte, contentType string, reqTimeStamp string) {
		// ensure tmp directory
		if err := os.MkdirAll("./tmp", 0755); err != nil {
			log.Printf("Error creating tmp dir: %v", err)
			return
		}

		if strings.HasPrefix(contentType, "application/json") {
			// JSON payload
			jsonFileName := fmt.Sprintf("%d.json", time.Now().UnixNano())
			destinationFilePath := filepath.Join("tmp", jsonFileName)
			os.WriteFile(destinationFilePath, body, 0644)
			log.Printf("Saved %s a JSON file, reqTime: %s", destinationFilePath, reqTimeStamp)

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
				destinationFilePath := filepath.Join("tmp", receivedFileName)
				f, err := os.Create(destinationFilePath)
				if err != nil {
					log.Printf("Error creating file: %v", err)
					continue
				}
				io.Copy(f, part)
				f.Close()
				fileCount++
			}
			log.Printf("Saved %d file(s) from multipart body, reqTime: %s", fileCount, reqTimeStamp)

		} else {
			// other payloads
			fname := fmt.Sprintf("%d.bin", time.Now().UnixNano())
			path := filepath.Join("tmp", fname)
			os.WriteFile(path, body, 0644)
		}
	}(bodyBytes, contentType, reqTime)

	// Log and respond
	log.Printf("[%s] %s request, payload size: %d bytes", reqTime, method, size)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	http.HandleFunc("/depot", depotHandler)
	log.Println("Server listening on :3003")
	if err := http.ListenAndServe(":3003", nil); err != nil {
		log.Fatal(err)
	}
}
