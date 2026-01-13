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
	// Try Qdrant if enabled
	if config.QdrantEnabled {
		backend, err := NewQdrantBackend(ctx, config)
		if err != nil {
			// Log warning and fall back to JSON
			fmt.Fprintf(os.Stderr, "Warning: Qdrant unavailable (%v), using JSON backend\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Using Qdrant backend with Ollama embeddings (semantic search)\n")
			return backend, nil
		}
	}

	// Use JSON backend
	fmt.Fprintf(os.Stderr, "Using JSON backend (keyword search)\n")
	return NewJSONBackend(config)
}

// isQdrantAvailable checks if Qdrant is accessible (deprecated)
func isQdrantAvailable(ctx context.Context, address string) bool {
	// This function is now deprecated - availability is checked in NewQdrantBackend
	return true
}
