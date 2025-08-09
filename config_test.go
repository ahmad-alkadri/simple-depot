package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
	}{
		{
			name:    "default values when no env vars set",
			envVars: map[string]string{},
			expected: &Config{
				ServerPort:     "3003",
				MinioEndpoint:  "minio:9000", // Updated to match dev container environment
				MinioAccessKey: "minioadmin",
				MinioSecretKey: "minioadmin",
				MinioBucket:    "depot-payloads",
				MinioUseSSL:    false,
			},
		},
		{
			name: "custom values from env vars",
			envVars: map[string]string{
				"SERVER_PORT":      "8080",
				"MINIO_ENDPOINT":   "minio.example.com:9000",
				"MINIO_ACCESS_KEY": "customkey",
				"MINIO_SECRET_KEY": "customsecret",
				"MINIO_BUCKET":     "custom-bucket",
				"MINIO_USE_SSL":    "true",
			},
			expected: &Config{
				ServerPort:     "8080",
				MinioEndpoint:  "minio.example.com:9000",
				MinioAccessKey: "customkey",
				MinioSecretKey: "customsecret",
				MinioBucket:    "custom-bucket",
				MinioUseSSL:    true,
			},
		},
		{
			name: "partial env vars with defaults",
			envVars: map[string]string{
				"SERVER_PORT":   "9090",
				"MINIO_USE_SSL": "true",
				"MINIO_BUCKET":  "test-bucket",
			},
			expected: &Config{
				ServerPort:     "9090",
				MinioEndpoint:  "minio:9000",
				MinioAccessKey: "minioadmin",
				MinioSecretKey: "minioadmin",
				MinioBucket:    "test-bucket",
				MinioUseSSL:    true,
			},
		},
		{
			name: "SSL false with explicit false value",
			envVars: map[string]string{
				"MINIO_USE_SSL": "false",
			},
			expected: &Config{
				ServerPort:     "3003",
				MinioEndpoint:  "minio:9000",
				MinioAccessKey: "minioadmin",
				MinioSecretKey: "minioadmin",
				MinioBucket:    "depot-payloads",
				MinioUseSSL:    false,
			},
		},
		{
			name: "SSL false with invalid value",
			envVars: map[string]string{
				"MINIO_USE_SSL": "invalid",
			},
			expected: &Config{
				ServerPort:     "3003",
				MinioEndpoint:  "minio:9000",
				MinioAccessKey: "minioadmin",
				MinioSecretKey: "minioadmin",
				MinioBucket:    "depot-payloads",
				MinioUseSSL:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			originalEnv := make(map[string]string)
			for key, value := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
				os.Setenv(key, value)
			}

			// Cleanup environment variables after test
			t.Cleanup(func() {
				for key := range tt.envVars {
					if originalValue, exists := originalEnv[key]; exists {
						os.Setenv(key, originalValue)
					} else {
						os.Unsetenv(key)
					}
				}
			})

			// Test the function
			config := LoadConfig()

			// Verify all fields
			if config.ServerPort != tt.expected.ServerPort {
				t.Errorf("ServerPort: got %s, want %s", config.ServerPort, tt.expected.ServerPort)
			}
			if config.MinioEndpoint != tt.expected.MinioEndpoint {
				t.Errorf("MinioEndpoint: got %s, want %s", config.MinioEndpoint, tt.expected.MinioEndpoint)
			}
			if config.MinioAccessKey != tt.expected.MinioAccessKey {
				t.Errorf("MinioAccessKey: got %s, want %s", config.MinioAccessKey, tt.expected.MinioAccessKey)
			}
			if config.MinioSecretKey != tt.expected.MinioSecretKey {
				t.Errorf("MinioSecretKey: got %s, want %s", config.MinioSecretKey, tt.expected.MinioSecretKey)
			}
			if config.MinioBucket != tt.expected.MinioBucket {
				t.Errorf("MinioBucket: got %s, want %s", config.MinioBucket, tt.expected.MinioBucket)
			}
			if config.MinioUseSSL != tt.expected.MinioUseSSL {
				t.Errorf("MinioUseSSL: got %t, want %t", config.MinioUseSSL, tt.expected.MinioUseSSL)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_ENV_VAR",
			defaultValue: "default",
			envValue:     "custom",
			setEnv:       true,
			expected:     "custom",
		},
		{
			name:         "returns default when env not set",
			key:          "UNSET_ENV_VAR",
			defaultValue: "default",
			envValue:     "",
			setEnv:       false,
			expected:     "default",
		},
		{
			name:         "returns empty string when env set to empty",
			key:          "EMPTY_ENV_VAR",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			expected:     "default", // getEnv treats empty string as not set
		},
		{
			name:         "returns whitespace value when set",
			key:          "WHITESPACE_ENV_VAR",
			defaultValue: "default",
			envValue:     "   ",
			setEnv:       true,
			expected:     "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalValue := os.Getenv(tt.key)
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			// Cleanup after test
			t.Cleanup(func() {
				if originalValue != "" {
					os.Setenv(tt.key, originalValue)
				} else {
					os.Unsetenv(tt.key)
				}
			})

			// Test the function
			result := getEnv(tt.key, tt.defaultValue)

			if result != tt.expected {
				t.Errorf("got %s, want %s", result, tt.expected)
			}
		})
	}
}
