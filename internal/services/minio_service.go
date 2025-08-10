package services

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ahmad-alkadri/simple-depot/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioService struct {
	client *minio.Client
	bucket string
}

// NewMinioService creates a new MinIO service
func NewMinioService(config *config.Config) (*MinioService, error) {
	// Initialize MinIO client
	client, err := minio.New(config.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.MinioAccessKey, config.MinioSecretKey, ""),
		Secure: config.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %v", err)
	}

	service := &MinioService{
		client: client,
		bucket: config.MinioBucket,
	}

	// Create bucket if it doesn't exist
	if err := service.ensureBucket(); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %v", err)
	}

	return service, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (m *MinioService) ensureBucket() error {
	ctx := context.Background()

	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("error checking if bucket exists: %v", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("error creating bucket: %v", err)
		}
		log.Printf("Created bucket: %s", m.bucket)
	}

	return nil
}

// SavePayload saves a payload to MinIO with the appropriate content type
func (m *MinioService) SavePayload(objectName string, data []byte, contentType string) error {
	ctx := context.Background()

	reader := bytes.NewReader(data)

	// Set appropriate content type if not provided
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Determine content type based on file extension if needed
	if strings.HasSuffix(objectName, ".json") {
		contentType = "application/json"
	} else if strings.HasSuffix(objectName, ".bin") {
		contentType = "application/octet-stream"
	}

	options := minio.PutObjectOptions{
		ContentType: contentType,
	}

	_, err := m.client.PutObject(ctx, m.bucket, objectName, reader, int64(len(data)), options)
	if err != nil {
		return fmt.Errorf("failed to upload object %s: %v", objectName, err)
	}

	log.Printf("Successfully saved payload to MinIO: %s (size: %d bytes)", objectName, len(data))
	return nil
}

// GetPayload retrieves a payload from MinIO
func (m *MinioService) GetPayload(objectName string) ([]byte, error) {
	ctx := context.Background()

	object, err := m.client.GetObject(ctx, m.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %v", objectName, err)
	}
	defer object.Close()

	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %s: %v", objectName, err)
	}

	return buffer.Bytes(), nil
}

// ListPayloads lists all payloads in the bucket
func (m *MinioService) ListPayloads() ([]string, error) {
	ctx := context.Background()

	var objects []string

	objectCh := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %v", object.Err)
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

// DeletePayload removes a payload from MinIO
func (m *MinioService) DeletePayload(objectName string) error {
	ctx := context.Background()

	err := m.client.RemoveObject(ctx, m.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %v", objectName, err)
	}

	log.Printf("Successfully deleted payload from MinIO: %s", objectName)
	return nil
}
