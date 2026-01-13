package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaEmbedder generates embeddings using Ollama's embedding API
type OllamaEmbedder struct {
	baseURL string
	model   string
	client  *http.Client
}

// OllamaEmbedRequest represents the request to Ollama's embedding API
type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbedResponse represents the response from Ollama's embedding API
type OllamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewOllamaEmbedder creates a new Ollama-based embedder
func NewOllamaEmbedder(baseURL, model string) *OllamaEmbedder {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "all-minilm:l6-v2" // 384-dim embeddings, same as Qdrant FastEmbed
	}

	return &OllamaEmbedder{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Embed generates an embedding for a single text
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	req := OllamaEmbedRequest{
		Model:  e.model,
		Prompt: text,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/api/embeddings", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response OllamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *OllamaEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		embedding, err := e.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// IsAvailable checks if Ollama is running and the model is available
func (e *OllamaEmbedder) IsAvailable(ctx context.Context) bool {
	// Try to generate a test embedding
	_, err := e.Embed(ctx, "test")
	return err == nil
}

// GetDimensions returns the embedding dimension for the model
func (e *OllamaEmbedder) GetDimensions() int {
	// all-minilm:l6-v2 produces 384-dimensional embeddings
	return 384
}
