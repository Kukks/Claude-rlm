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
	RAGDir string
}

// DefaultConfig returns default storage configuration
func DefaultConfig(ragDir string) *Config {
	return &Config{
		RAGDir: ragDir,
	}
}

// NewBackend creates a storage backend
func NewBackend(ctx context.Context, config *Config) (Backend, error) {
	// Use BM25 backend (pure Go, zero external dependencies)
	backend, err := NewBM25Backend(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize BM25 backend: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Using BM25 backend (pure Go search, zero dependencies)\n")
	return backend, nil
}
