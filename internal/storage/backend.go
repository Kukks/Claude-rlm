package storage

import (
	"context"
	"fmt"
	"os"
)

// Backend defines the interface for storage implementations
type Backend interface {
	// Store saves an analysis result
	Store(ctx context.Context, data *AnalysisData) error

	// Search performs a search query
	Search(ctx context.Context, query string, limit int) ([]*SearchResult, error)

	// GetAll retrieves all stored analyses
	GetAll(ctx context.Context) ([]*AnalysisData, error)

	// Close cleans up resources
	Close() error

	// Name returns the backend name
	Name() string
}

// Config holds storage configuration
type Config struct {
	RAGDir         string
	QdrantAddress  string
	QdrantEnabled  bool
	CollectionName string
}

// DefaultConfig returns default storage configuration
func DefaultConfig(ragDir string) *Config {
	return &Config{
		RAGDir:         ragDir,
		QdrantAddress:  "localhost:6334",
		QdrantEnabled:  true,
		CollectionName: "rlm_analyses",
	}
}

// NewBackend creates a storage backend with automatic fallback
func NewBackend(ctx context.Context, config *Config) (Backend, error) {
	// Try Qdrant first if enabled
	if config.QdrantEnabled {
		if isQdrantAvailable(ctx, config.QdrantAddress) {
			backend, err := NewQdrantBackend(ctx, config)
			if err == nil {
				return backend, nil
			}
			// Log warning but continue to fallback
			fmt.Fprintf(os.Stderr, "Warning: Qdrant unavailable (%v), falling back to JSON\n", err)
		}
	}

	// Fallback to JSON backend
	return NewJSONBackend(config)
}

// isQdrantAvailable checks if Qdrant is accessible
func isQdrantAvailable(ctx context.Context, address string) bool {
	// Try to connect to Qdrant
	// For now, we'll implement actual connectivity check in qdrant.go
	// This is just a placeholder
	return true
}
