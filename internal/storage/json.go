package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// JSONBackend implements storage using JSON files with keyword search
type JSONBackend struct {
	ragDir string
}

// NewJSONBackend creates a new JSON storage backend
func NewJSONBackend(config *Config) (*JSONBackend, error) {
	// Ensure RAG directory exists
	if err := os.MkdirAll(config.RAGDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create RAG directory: %w", err)
	}

	return &JSONBackend{
		ragDir: config.RAGDir,
	}, nil
}

// Store saves an analysis result to JSON files
func (j *JSONBackend) Store(ctx context.Context, data *AnalysisData) error {
	// Generate ID if not provided
	if data.ID == "" {
		data.ID = uuid.New().String()
	}

	data.Backend = "json"
	data.Version = "3.0"

	// Save full data to JSON file
	if err := j.saveJSONFile(data); err != nil {
		return fmt.Errorf("failed to save JSON file: %w", err)
	}

	// Update index
	if err := j.updateIndex(data); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	return nil
}

// Search performs keyword-based search
func (j *JSONBackend) Search(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	// Load all entries
	entries, err := j.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	// Score each entry based on keyword matching
	type scoredResult struct {
		data  *AnalysisData
		score float64
	}

	queryWords := strings.Fields(strings.ToLower(query))
	querySet := make(map[string]bool)
	for _, word := range queryWords {
		querySet[word] = true
	}

	scored := make([]scoredResult, 0, len(entries))
	for _, entry := range entries {
		score := j.calculateScore(query, queryWords, querySet, entry)
		if score > 0 {
			scored = append(scored, scoredResult{data: entry, score: score})
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Take top N results
	if len(scored) > limit {
		scored = scored[:limit]
	}

	// Convert to SearchResult
	results := make([]*SearchResult, len(scored))
	for i, sr := range scored {
		results[i] = &SearchResult{
			Data:         sr.data,
			Score:        sr.score,
			SearchMethod: "keyword",
		}
	}

	return results, nil
}

// calculateScore computes relevance score for keyword matching
func (j *JSONBackend) calculateScore(query string, queryWords []string, querySet map[string]bool, data *AnalysisData) float64 {
	score := 0.0

	entryQuery := strings.ToLower(data.Query)
	entryFocus := strings.ToLower(data.Focus)

	// Exact match in query: +10
	if strings.Contains(entryQuery, strings.ToLower(query)) {
		score += 10.0
	}

	// Exact match in focus: +5
	if strings.Contains(entryFocus, strings.ToLower(query)) {
		score += 5.0
	}

	// Word overlap scoring: +2 per matching word
	entryWords := strings.Fields(entryQuery + " " + entryFocus)
	commonWords := 0
	for _, word := range entryWords {
		if querySet[strings.ToLower(word)] {
			commonWords++
		}
	}
	score += float64(commonWords) * 2.0

	return score
}

// GetAll retrieves all stored analyses
func (j *JSONBackend) GetAll(ctx context.Context) ([]*AnalysisData, error) {
	// Read index file
	index, err := j.loadIndex()
	if err != nil {
		if os.IsNotExist(err) {
			return []*AnalysisData{}, nil
		}
		return nil, err
	}

	results := make([]*AnalysisData, 0, len(index))
	for _, entry := range index {
		data, err := j.loadJSONFile(entry.ID)
		if err != nil {
			continue // Skip if file not found
		}
		results = append(results, data)
	}

	return results, nil
}

// Close cleans up resources (no-op for JSON backend)
func (j *JSONBackend) Close() error {
	return nil
}

// Name returns the backend name
func (j *JSONBackend) Name() string {
	return "json"
}

// saveJSONFile saves the full analysis data to a JSON file
func (j *JSONBackend) saveJSONFile(data *AnalysisData) error {
	filename := filepath.Join(j.ragDir, fmt.Sprintf("analysis_%s.json", data.ID))

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0644)
}

// loadJSONFile loads the full analysis data from a JSON file
func (j *JSONBackend) loadJSONFile(id string) (*AnalysisData, error) {
	filename := filepath.Join(j.ragDir, fmt.Sprintf("analysis_%s.json", id))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var analysis AnalysisData
	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, err
	}

	return &analysis, nil
}

// updateIndex adds an entry to the index
func (j *JSONBackend) updateIndex(data *AnalysisData) error {
	index, err := j.loadIndex()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Add new entry
	entry := IndexEntry{
		ID:             data.ID,
		Query:          data.Query,
		Focus:          data.Focus,
		Timestamp:      data.Timestamp,
		Path:           data.Path,
		HasVectorEmbed: false,
		StorageBackend: "json",
	}

	index = append(index, entry)

	// Save index
	indexFile := filepath.Join(j.ragDir, "index.json")
	indexData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexFile, indexData, 0644)
}

// loadIndex loads the index from disk
func (j *JSONBackend) loadIndex() ([]IndexEntry, error) {
	indexFile := filepath.Join(j.ragDir, "index.json")

	data, err := os.ReadFile(indexFile)
	if err != nil {
		return nil, err
	}

	var index []IndexEntry
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	return index, nil
}
