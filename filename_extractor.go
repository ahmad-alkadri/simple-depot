package main

import (
	"path/filepath"
	"strings"
)

// DefaultFilenameExtractor extracts filenames from HTTP headers
type DefaultFilenameExtractor struct{}

// NewDefaultFilenameExtractor creates a new filename extractor
func NewDefaultFilenameExtractor() *DefaultFilenameExtractor {
	return &DefaultFilenameExtractor{}
}

// Extract extracts filename from Content-Disposition header
func (e *DefaultFilenameExtractor) Extract(contentDisposition string) string {
	if contentDisposition == "" {
		return ""
	}

	if idx := strings.Index(contentDisposition, "filename="); idx != -1 {
		start := idx + 9 // len("filename=")
		filename := contentDisposition[start:]
		filename = strings.Trim(filename, "\"")
		return filepath.Base(filename)
	}

	return ""
}
