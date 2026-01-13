package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/kukks/claude-rlm/internal/embeddings"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantBackend implements storage using Qdrant vector database
type QdrantBackend struct {
	client     *qdrant.Client
	embedder   *embeddings.OllamaEmbedder
	collection string
	ragDir     string
}

// NewQdrantBackend creates a new Qdrant storage backend
func NewQdrantBackend(ctx context.Context, config *Config) (*QdrantBackend, error) {
	// Parse host and port from address
	host := "localhost"
	port := 6334
	// Note: Could parse config.QdrantAddress to extract host:port if needed

	// Create Ollama embedder
	embedder := embeddings.NewOllamaEmbedder("", "") // Use defaults

	// Check if Ollama is available
	if !embedder.IsAvailable(ctx) {
		return nil, fmt.Errorf("Ollama is not available - install Ollama and run: ollama pull all-minilm:l6-v2")
	}

	// Connect to Qdrant
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	backend := &QdrantBackend{
		client:     client,
		embedder:   embedder,
		collection: config.CollectionName,
		ragDir:     config.RAGDir,
	}

	// Ensure collection exists
	if err := backend.ensureCollection(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	// Ensure RAG directory exists
	if err := os.MkdirAll(config.RAGDir, 0755); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create RAG directory: %w", err)
	}

	return backend, nil
}

// ensureCollection creates the collection if it doesn't exist
func (q *QdrantBackend) ensureCollection(ctx context.Context) error {
	// List existing collections (returns []string)
	collections, err := q.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	// Check if our collection exists
	for _, colName := range collections {
		if colName == q.collection {
			return nil // Collection already exists
		}
	}

	// Create collection with 384-dimensional vectors (all-minilm:l6-v2 output)
	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: q.collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(q.embedder.GetDimensions()),
			Distance: qdrant.Distance_Cosine,
		}),
	})

	return err
}

// Store saves an analysis result to Qdrant and JSON
func (q *QdrantBackend) Store(ctx context.Context, data *AnalysisData) error {
	// Generate ID if not provided
	if data.ID == "" {
		data.ID = uuid.New().String()
	}

	data.Backend = "qdrant"
	data.Version = "3.0"

	// Create content string for embedding
	contentJSON, _ := json.Marshal(data.Result)
	content := fmt.Sprintf("Query: %s\nFocus: %s\nResult: %s", data.Query, data.Focus, string(contentJSON))

	// Generate embedding using Ollama (returns []float32)
	embedding, err := q.embedder.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Prepare metadata payload
	metadata := map[string]any{
		"id":        data.ID,
		"query":     data.Query,
		"focus":     data.Focus,
		"timestamp": data.Timestamp.Format("2006-01-02T15:04:05Z"),
		"path":      data.Path,
	}

	// Upsert to Qdrant
	_, err = q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collection,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(uint64(uuid.MustParse(data.ID).ID())), // Use UUID numeric ID
				Vectors: qdrant.NewVectors(embedding...),
				Payload: qdrant.NewValueMap(metadata),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upsert to Qdrant: %w", err)
	}

	// Also save full data to JSON file for complete retrieval
	if err := q.saveJSONFile(data); err != nil {
		return fmt.Errorf("failed to save JSON file: %w", err)
	}

	// Update index
	if err := q.updateIndex(data); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	return nil
}

// Search performs semantic search using Qdrant
func (q *QdrantBackend) Search(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	// Generate embedding for query (returns []float32)
	queryEmbedding, err := q.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search Qdrant
	searchResult, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.collection,
		Query:          qdrant.NewQuery(queryEmbedding...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("Qdrant search failed: %w", err)
	}

	// Load full data from JSON files
	results := make([]*SearchResult, 0, len(searchResult))
	for _, point := range searchResult {
		// Get ID from payload
		id := point.GetPayload()["id"].GetStringValue()

		// Load full data from JSON
		data, err := q.loadJSONFile(id)
		if err != nil {
			continue // Skip if JSON file not found
		}

		// Convert score to 0-100 (Qdrant returns 0-1 for cosine similarity)
		score := float64(point.GetScore()) * 100.0

		results = append(results, &SearchResult{
			Data:         data,
			Score:        score,
			SearchMethod: "semantic",
		})
	}

	return results, nil
}

// GetAll retrieves all stored analyses
func (q *QdrantBackend) GetAll(ctx context.Context) ([]*AnalysisData, error) {
	// Read index file
	index, err := q.loadIndex()
	if err != nil {
		return nil, err
	}

	results := make([]*AnalysisData, 0, len(index))
	for _, entry := range index {
		data, err := q.loadJSONFile(entry.ID)
		if err != nil {
			continue
		}
		results = append(results, data)
	}

	return results, nil
}

// Close cleans up resources
func (q *QdrantBackend) Close() error {
	if q.client != nil {
		return q.client.Close()
	}
	return nil
}

// Name returns the backend name
func (q *QdrantBackend) Name() string {
	return "qdrant"
}

// saveJSONFile saves the full analysis data to a JSON file
func (q *QdrantBackend) saveJSONFile(data *AnalysisData) error {
	filename := filepath.Join(q.ragDir, fmt.Sprintf("analysis_%s.json", data.ID))

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0644)
}

// loadJSONFile loads the full analysis data from a JSON file
func (q *QdrantBackend) loadJSONFile(id string) (*AnalysisData, error) {
	filename := filepath.Join(q.ragDir, fmt.Sprintf("analysis_%s.json", id))

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
func (q *QdrantBackend) updateIndex(data *AnalysisData) error {
	index, err := q.loadIndex()
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
		HasVectorEmbed: true,
		StorageBackend: "qdrant",
	}

	index = append(index, entry)

	// Save index
	indexFile := filepath.Join(q.ragDir, "index.json")
	indexData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexFile, indexData, 0644)
}

// loadIndex loads the index from disk
func (q *QdrantBackend) loadIndex() ([]IndexEntry, error) {
	indexFile := filepath.Join(q.ragDir, "index.json")

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
