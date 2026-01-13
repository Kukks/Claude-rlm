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
	// TODO: Qdrant backend requires embedding model integration
	// The go-client doesn't support server-side FastEmbed text embeddings
	// For now, always use JSON backend with keyword search
	if config.QdrantEnabled {
		fmt.Fprintf(os.Stderr, "Note: Qdrant support pending embedding integration, using JSON backend\n")
	}

	// Use JSON backend
	return NewJSONBackend(config)
}

// isQdrantAvailable checks if Qdrant is accessible
func isQdrantAvailable(ctx context.Context, address string) bool {
	// Try to connect to Qdrant
	// For now, we'll implement actual connectivity check in qdrant.go
	// This is just a placeholder
	return true
}
