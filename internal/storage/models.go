package storage

import (
	"time"

	"github.com/kukks/claude-rlm/internal/orchestrator"
)

// AnalysisData represents a complete analysis entry
type AnalysisData struct {
	ID          string                 `json:"id"`
	Query       string                 `json:"query"`
	Focus       string                 `json:"focus"`
	Timestamp   time.Time              `json:"timestamp"`
	Result      map[string]interface{} `json:"result"`
	Stats       orchestrator.Stats     `json:"stats"`
	Path        string                 `json:"path"`
	FileHashes  map[string]string      `json:"file_hashes"`
	Version     string                 `json:"version"`
	Backend     string                 `json:"storage_backend"`
}

// SearchResult wraps an analysis result with a relevance score
type SearchResult struct {
	Data         *AnalysisData `json:"data"`
	Score        float64       `json:"score"`
	SearchMethod string        `json:"search_method"` // "semantic" or "keyword"
}

// IndexEntry is a lightweight entry in the index
type IndexEntry struct {
	ID               string    `json:"id"`
	Query            string    `json:"query"`
	Focus            string    `json:"focus"`
	Timestamp        time.Time `json:"timestamp"`
	Path             string    `json:"path"`
	HasVectorEmbed   bool      `json:"has_vector_embedding"`
	StorageBackend   string    `json:"storage_backend"`
}
