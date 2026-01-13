package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/crawlab-team/bm25"
	"github.com/google/uuid"
)

// BM25Backend implements storage using BM25 search algorithm
type BM25Backend struct {
	ragDir string
	index  bm25.BM25 // BM25 interface
	corpus []string  // Document corpus for BM25
	docIDs []string  // Document IDs corresponding to corpus
	mu     sync.RWMutex
}

// NewBM25Backend creates a new BM25 storage backend
func NewBM25Backend(config *Config) (*BM25Backend, error) {
	// Ensure RAG directory exists
	if err := os.MkdirAll(config.RAGDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create RAG directory: %w", err)
	}

	backend := &BM25Backend{
		ragDir: config.RAGDir,
		corpus: make([]string, 0),
		docIDs: make([]string, 0),
	}

	// Load existing documents from disk
	if err := backend.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load existing documents: %w", err)
	}

	return backend, nil
}

// Store saves an analysis result with BM25 indexing
func (b *BM25Backend) Store(ctx context.Context, data *AnalysisData) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Generate ID if not provided
	if data.ID == "" {
		data.ID = uuid.New().String()
	}

	data.Backend = "bm25"
	data.Version = "3.0"

	// Create searchable content string
	contentJSON, _ := json.Marshal(data.Result)
	content := fmt.Sprintf("%s %s %s", data.Query, data.Focus, string(contentJSON))

	// Add to corpus
	b.corpus = append(b.corpus, content)
	b.docIDs = append(b.docIDs, data.ID)

	// Rebuild BM25 index with new corpus
	b.rebuildIndex()

	// Save full data to JSON file
	if err := b.saveJSONFile(data); err != nil {
		return fmt.Errorf("failed to save JSON file: %w", err)
	}

	// Update index
	if err := b.updateIndex(data); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	return nil
}

// Search performs BM25-based search
func (b *BM25Backend) Search(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.corpus) == 0 {
		return []*SearchResult{}, nil
	}

	// Tokenize query
	queryTokens := tokenize(query)

	// Get BM25 scores for all documents
	scores, err := b.index.GetScores(queryTokens)
	if err != nil {
		return nil, fmt.Errorf("BM25 scoring failed: %w", err)
	}

	// Create scored results
	type scoredResult struct {
		id    string
		score float64
	}
	scoredResults := make([]scoredResult, 0, len(scores))
	for i, score := range scores {
		if score > 0 { // Only include results with positive scores
			scoredResults = append(scoredResults, scoredResult{
				id:    b.docIDs[i],
				score: score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(scoredResults, func(i, j int) bool {
		return scoredResults[i].score > scoredResults[j].score
	})

	// Limit results
	if limit > 0 && len(scoredResults) > limit {
		scoredResults = scoredResults[:limit]
	}

	// Load full data for top results
	results := make([]*SearchResult, 0, len(scoredResults))
	for _, sr := range scoredResults {
		data, err := b.loadJSONFile(sr.id)
		if err != nil {
			continue // Skip if JSON file not found
		}

		// Normalize score to 0-100 range
		normalizedScore := normalizeScore(sr.score)

		results = append(results, &SearchResult{
			Data:         data,
			Score:        normalizedScore,
			SearchMethod: "bm25",
		})
	}

	return results, nil
}

// GetAll retrieves all stored analyses
func (b *BM25Backend) GetAll(ctx context.Context) ([]*AnalysisData, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Read index file
	index, err := b.loadIndexFile()
	if err != nil {
		return nil, err
	}

	results := make([]*AnalysisData, 0, len(index))
	for _, entry := range index {
		data, err := b.loadJSONFile(entry.ID)
		if err != nil {
			continue
		}
		results = append(results, data)
	}

	return results, nil
}

// Close cleans up resources
func (b *BM25Backend) Close() error {
	// BM25 backend has no persistent connections
	return nil
}

// Name returns the backend name
func (b *BM25Backend) Name() string {
	return "bm25"
}

// rebuildIndex rebuilds the BM25 index from current corpus
func (b *BM25Backend) rebuildIndex() {
	if len(b.corpus) == 0 {
		return
	}

	// Create new BM25 index using Okapi variant
	// Parameters: k1=1.5, b=0.75 (standard BM25 parameters), logger=nil
	index, err := bm25.NewBM25Okapi(b.corpus, tokenize, 1.5, 0.75, nil)
	if err != nil {
		// Log error but don't fail - we'll continue with nil index
		fmt.Fprintf(os.Stderr, "Warning: Failed to build BM25 index: %v\n", err)
		return
	}
	b.index = index
}

// loadFromDisk loads existing documents from disk and rebuilds index
func (b *BM25Backend) loadFromDisk() error {
	index, err := b.loadIndexFile()
	if err != nil {
		if os.IsNotExist(err) {
			// No existing index, start fresh
			b.rebuildIndex()
			return nil
		}
		return err
	}

	// Load documents and build corpus
	for _, entry := range index {
		data, err := b.loadJSONFile(entry.ID)
		if err != nil {
			continue // Skip missing files
		}

		// Recreate searchable content
		contentJSON, _ := json.Marshal(data.Result)
		content := fmt.Sprintf("%s %s %s", data.Query, data.Focus, string(contentJSON))

		b.corpus = append(b.corpus, content)
		b.docIDs = append(b.docIDs, data.ID)
	}

	// Rebuild index with loaded corpus
	b.rebuildIndex()

	return nil
}

// saveJSONFile saves the full analysis data to a JSON file
func (b *BM25Backend) saveJSONFile(data *AnalysisData) error {
	filename := filepath.Join(b.ragDir, fmt.Sprintf("analysis_%s.json", data.ID))

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0644)
}

// loadJSONFile loads the full analysis data from a JSON file
func (b *BM25Backend) loadJSONFile(id string) (*AnalysisData, error) {
	filename := filepath.Join(b.ragDir, fmt.Sprintf("analysis_%s.json", id))

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
func (b *BM25Backend) updateIndex(data *AnalysisData) error {
	index, err := b.loadIndexFile()
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
		HasVectorEmbed: false, // BM25 doesn't use embeddings
		StorageBackend: "bm25",
	}

	index = append(index, entry)

	// Save index
	indexFile := filepath.Join(b.ragDir, "index.json")
	indexData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexFile, indexData, 0644)
}

// loadIndexFile loads the index from disk
func (b *BM25Backend) loadIndexFile() ([]IndexEntry, error) {
	indexFile := filepath.Join(b.ragDir, "index.json")

	data, err := os.ReadFile(indexFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []IndexEntry{}, nil
		}
		return nil, err
	}

	var index []IndexEntry
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	return index, nil
}

// tokenize splits text into tokens for BM25
func tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Split on whitespace and punctuation
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})

	// Filter out very short tokens
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if len(token) >= 2 { // Keep tokens with 2+ characters
			filtered = append(filtered, token)
		}
	}

	return filtered
}

// normalizeScore normalizes BM25 score to 0-100 range
func normalizeScore(score float64) float64 {
	// BM25 scores are unbounded, but typically 0-10 for good matches
	// We'll map 0-10 to 0-100, capping at 100
	normalized := score * 10.0
	if normalized > 100.0 {
		normalized = 100.0
	}
	return normalized
}
