package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
)

// QdrantBackend implements storage using Qdrant vector database
type QdrantBackend struct {
	client         pb.QdrantClient
	collectionName string
	ragDir         string
}

// NewQdrantBackend creates a new Qdrant storage backend
func NewQdrantBackend(ctx context.Context, config *Config) (*QdrantBackend, error) {
	// Connect to Qdrant
	conn, err := pb.NewQdrantClient(ctx, &pb.Config{
		Address: config.QdrantAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	backend := &QdrantBackend{
		client:         conn,
		collectionName: config.CollectionName,
		ragDir:         config.RAGDir,
	}

	// Ensure collection exists
	if err := backend.ensureCollection(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	// Ensure RAG directory exists
	if err := os.MkdirAll(config.RAGDir, 0755); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create RAG directory: %w", err)
	}

	return backend, nil
}

// ensureCollection creates the collection if it doesn't exist
func (q *QdrantBackend) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	collections, err := q.client.ListCollections(ctx)
	if err != nil {
		return err
	}

	for _, col := range collections.GetCollections() {
		if col.GetName() == q.collectionName {
			return nil // Collection already exists
		}
	}

	// Create collection with text embeddings
	// Qdrant's FastEmbed uses 384-dimensional vectors (all-MiniLM-L6-v2)
	_, err = q.client.CreateCollection(ctx, &pb.CreateCollection{
		CollectionName: q.collectionName,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     384,
					Distance: pb.Distance_Cosine,
				},
			},
		},
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

	// Prepare metadata payload
	metadata := map[string]interface{}{
		"id":        data.ID,
		"query":     data.Query,
		"focus":     data.Focus,
		"timestamp": data.Timestamp.Format("2006-01-02T15:04:05Z"),
		"path":      data.Path,
	}

	// Convert metadata to Qdrant payload
	payload, err := pb.NewStruct(metadata)
	if err != nil {
		return fmt.Errorf("failed to create payload: %w", err)
	}

	// Upsert to Qdrant with text (Qdrant will embed it automatically using FastEmbed)
	point := &pb.PointStruct{
		Id: &pb.PointId{
			PointIdOptions: &pb.PointId_Uuid{
				Uuid: data.ID,
			},
		},
		Vectors: &pb.Vectors{
			VectorsOptions: &pb.Vectors_Text{
				Text: content,
			},
		},
		Payload: payload,
	}

	_, err = q.client.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: q.collectionName,
		Points:         []*pb.PointStruct{point},
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
	// Search Qdrant using text query (FastEmbed will embed it)
	searchResult, err := q.client.Search(ctx, &pb.SearchPoints{
		CollectionName: q.collectionName,
		Limit:          uint64(limit),
		Vector: &pb.Vector{
			Data: &pb.Vector_Text{
				Text: query,
			},
		},
		WithPayload: &pb.WithPayloadSelector{
			SelectorOptions: &pb.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
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

		// Convert distance to score (0-100, higher is better)
		// Cosine distance: 0 = identical, 2 = opposite
		// Convert to score: 100 - (distance * 50)
		score := 100.0 - (float64(point.GetScore()) * 50.0)
		if score < 0 {
			score = 0
		}

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
	return q.client.Close()
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
