package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// DefaultIDGenerator generates unique IDs using timestamp and random bytes
type DefaultIDGenerator struct{}

// NewDefaultIDGenerator creates a new default ID generator
func NewDefaultIDGenerator() *DefaultIDGenerator {
	return &DefaultIDGenerator{}
}

// Generate creates a unique identifier
func (g *DefaultIDGenerator) Generate() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to nanoseconds if random fails
		return fmt.Sprintf("%d_%d", timestamp, time.Now().UnixNano())
	}
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%d_%s", timestamp, randomHex)
}
