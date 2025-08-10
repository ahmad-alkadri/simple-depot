package services

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
)

// DefaultZipService handles creating zip archives
type DefaultZipService struct{}

// NewDefaultZipService creates a new zip service
func NewDefaultZipService() *DefaultZipService {
	return &DefaultZipService{}
}

// CreateZip creates a zip archive from multiple files
func (z *DefaultZipService) CreateZip(files []FileInfo) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	defer zipWriter.Close()

	for _, file := range files {
		filename := file.OriginalFilename
		if filename == "" {
			filename = file.ObjectName
		}

		// Decode base64 data
		decoded, err := base64.StdEncoding.DecodeString(file.PayloadBase64)
		if err != nil {
			continue
		}

		// Create file in zip
		f, err := zipWriter.Create(filename)
		if err != nil {
			continue
		}

		_, err = f.Write(decoded)
		if err != nil {
			continue
		}
	}

	zipWriter.Close()
	return buf.Bytes(), nil
}
