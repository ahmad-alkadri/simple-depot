package config

import (
	"os"
	"sync"
	"time"
)

type Config struct {
	ServerPort     string
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
}

type ConfigManager struct {
	mu     sync.RWMutex
	config *Config
}

func NewConfigManager() *ConfigManager {
	cm := &ConfigManager{
		config: LoadConfig(),
	}
	go cm.periodicReload()
	return cm
}

func (cm *ConfigManager) periodicReload() {
	for {
		newConfig := LoadConfig()
		cm.mu.Lock()
		cm.config = newConfig
		cm.mu.Unlock()
		time.Sleep(10 * time.Second)
	}
}

func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:     GetEnv("SERVER_PORT", "3003"),
		MinioEndpoint:  GetEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: GetEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: GetEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    GetEnv("MINIO_BUCKET", "depot-payloads"),
		MinioUseSSL:    GetEnv("MINIO_USE_SSL", "false") == "true",
	}
}

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
