package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/kukks/claude-rlm/internal/orchestrator"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestratorCreation(t *testing.T) {
	config := orchestrator.DefaultConfig()
	logger := zerolog.Nop()

	orch := orchestrator.New(config, logger)
	assert.NotNil(t, orch)
}

func TestTrampolinePattern(t *testing.T) {
	config := orchestrator.DefaultConfig()
	logger := zerolog.Nop()

	orch := orchestrator.New(config, logger)

	// Set up a mock dispatcher that returns analysis result
	orch.SetDispatcher(func(ctx context.Context, task *orchestrator.Task) (*orchestrator.SubagentResult, error) {
		return &orchestrator.SubagentResult{
			Type: orchestrator.ResultTypeAnalysis,
			Analysis: &orchestrator.AnalysisResult{
				Type:       "RESULT",
				Content:    "Test analysis result",
				Metadata:   map[string]interface{}{},
				TokenCount: 100,
				CostUSD:    0.001,
			},
		}, nil
	})

	// Run analysis
	ctx := context.Background()
	result, err := orch.AnalyzeDocument(ctx, "test.txt", "test query")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test analysis result", result.Content)
	assert.Equal(t, 100, result.TokenCount)

	// Check stats
	stats := orch.GetStats()
	assert.Equal(t, 1, stats.TotalSubagentCalls)
	assert.Equal(t, 100, stats.TotalTokens)
}

func TestCaching(t *testing.T) {
	config := orchestrator.DefaultConfig()
	config.CacheEnabled = true
	config.CacheTTL = 1 * time.Hour
	logger := zerolog.Nop()

	orch := orchestrator.New(config, logger)

	callCount := 0
	orch.SetDispatcher(func(ctx context.Context, task *orchestrator.Task) (*orchestrator.SubagentResult, error) {
		callCount++
		return &orchestrator.SubagentResult{
			Type: orchestrator.ResultTypeAnalysis,
			Analysis: &orchestrator.AnalysisResult{
				Type:       "RESULT",
				Content:    "Cached result",
				Metadata:   map[string]interface{}{},
				TokenCount: 100,
				CostUSD:    0.001,
			},
		}, nil
	})

	ctx := context.Background()

	// First call - should hit dispatcher
	result1, err := orch.AnalyzeDocument(ctx, "test.txt", "cache test")
	require.NoError(t, err)
	assert.NotNil(t, result1)
	assert.Equal(t, 1, callCount)

	// Second call with same query - should use cache
	result2, err := orch.AnalyzeDocument(ctx, "test.txt", "cache test")
	require.NoError(t, err)
	assert.NotNil(t, result2)
	assert.Equal(t, 1, callCount) // Should still be 1 (cached)

	stats := orch.GetStats()
	assert.Equal(t, 1, stats.CacheHits)
}

func TestMaxDepthLimit(t *testing.T) {
	config := orchestrator.DefaultConfig()
	config.MaxRecursionDepth = 2
	logger := zerolog.Nop()

	orch := orchestrator.New(config, logger)

	// Set up dispatcher that always requests continuation
	orch.SetDispatcher(func(ctx context.Context, task *orchestrator.Task) (*orchestrator.SubagentResult, error) {
		return &orchestrator.SubagentResult{
			Type: orchestrator.ResultTypeContinuation,
			Continuation: &orchestrator.ContinuationRequest{
				Type:      "CONTINUATION",
				AgentType: "Worker",
				Task:      "deeper analysis",
				Context:   map[string]interface{}{},
				ReturnTo:  "test",
			},
		}, nil
	})

	ctx := context.Background()

	// Should fail due to max depth
	_, err := orch.AnalyzeDocument(ctx, "test.txt", "test query")
	assert.Error(t, err)
	assert.Equal(t, orchestrator.ErrMaxDepthExceeded, err)
}
