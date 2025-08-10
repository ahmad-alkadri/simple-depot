package main

import (
	"log"
	"net/http"
)

func main() {
	// Create ConfigManager
	configManager := NewConfigManager()
	config := configManager.GetConfig()
	log.Printf("Starting server with config: Endpoint=%s, Bucket=%s, UseSSL=%v",
		config.MinioEndpoint, config.MinioBucket, config.MinioUseSSL)

	// Initialize MinIO service
	var err error
	storageService, err := NewMinioService(config)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO service: %v", err)
	}

	log.Println("MinIO service initialized successfully")

	api := &DepotAPI{Storage: storageService}

	http.HandleFunc("/depot", api.DepotHandler)
	http.HandleFunc("/list", api.ListHandler)

	serverAddr := ":" + config.ServerPort
	log.Printf("Server listening on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Fatal(err)
	}
}
