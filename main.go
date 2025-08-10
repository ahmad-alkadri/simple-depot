package main

import (
	"log"
	"net/http"

	"github.com/ahmad-alkadri/simple-depot/internal/config"
	"github.com/ahmad-alkadri/simple-depot/internal/handlers"
	"github.com/ahmad-alkadri/simple-depot/internal/services"
)

func main() {
	// Create ConfigManager
	configManager := config.NewConfigManager()
	config := configManager.GetConfig()
	log.Printf("Starting server with config: Endpoint=%s, Bucket=%s, UseSSL=%v",
		config.MinioEndpoint, config.MinioBucket, config.MinioUseSSL)

	// Initialize storage service
	storageService, err := services.NewMinioService(config)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO service: %v", err)
	}
	log.Println("MinIO service initialized successfully")

	// Create all service dependencies (following dependency injection)
	idGenerator := services.NewDefaultIDGenerator()
	contentTypeDetector := services.NewDefaultContentTypeDetector()
	filenameExtractor := services.NewDefaultFilenameExtractor()
	responseFormatter := services.NewDefaultResponseFormatter()
	zipService := services.NewDefaultZipService()
	payloadProcessor := services.NewDefaultPayloadProcessor(contentTypeDetector)

	// Create payload service with all dependencies
	payloadService := services.NewDefaultPayloadService(
		storageService,
		payloadProcessor,
		idGenerator,
		responseFormatter,
		zipService,
	)

	// Create HTTP handler with dependencies
	httpHandler := handlers.NewHTTPHandler(payloadService, responseFormatter, filenameExtractor)

	// Setup routes
	http.HandleFunc("/depot", httpHandler.DepotHandler)
	http.HandleFunc("/list", httpHandler.ListHandler)
	http.HandleFunc("/get", httpHandler.GetHandler)

	serverAddr := ":" + config.ServerPort
	log.Printf("Server listening on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Fatal(err)
	}
}
