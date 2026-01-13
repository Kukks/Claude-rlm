package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// Config holds orchestrator configuration
type Config struct {
	MaxRecursionDepth int
	MaxIterations     int
	CacheEnabled      bool
	CacheTTL          time.Duration
	WorkDir           string
	StateFile         string
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxRecursionDepth: 10,
		MaxIterations:     1000,
		CacheEnabled:      true,
		CacheTTL:          DefaultCacheTTL,
		WorkDir:           ".",
		StateFile:         StateFileName,
	}
}

// Orchestrator manages the trampoline-based recursion pattern
type Orchestrator struct {
	config      *Config
	logger      zerolog.Logger
	stack       []Task
	currentTask Task
	results     map[string]interface{}
	stats       Stats
	dispatcher  SubagentDispatcher
}

// SubagentDispatcher is a function that dispatches work to a subagent
// Returns either a ContinuationRequest or AnalysisResult
type SubagentDispatcher func(ctx context.Context, task *Task) (*SubagentResult, error)

var (
	ErrMaxDepthExceeded      = errors.New("maximum recursion depth exceeded")
	ErrMaxIterationsExceeded = errors.New("maximum iterations exceeded")
	ErrNoDispatcher          = errors.New("no subagent dispatcher configured")
)

// New creates a new orchestrator
func New(config *Config, logger zerolog.Logger) *Orchestrator {
	if config == nil {
		config = DefaultConfig()
	}

	return &Orchestrator{
		config:  config,
		logger:  logger,
		stack:   make([]Task, 0),
		results: make(map[string]interface{}),
		stats: Stats{
			StartTime: time.Now(),
		},
	}
}

// SetDispatcher sets the subagent dispatcher function
func (o *Orchestrator) SetDispatcher(dispatcher SubagentDispatcher) {
	o.dispatcher = dispatcher
}

// Stats returns the current statistics
func (o *Orchestrator) GetStats() Stats {
	return o.stats
}

// AnalyzeDocument is the main entry point for document analysis
func (o *Orchestrator) AnalyzeDocument(ctx context.Context, documentPath, query string) (*AnalysisResult, error) {
	if o.dispatcher == nil {
		return nil, ErrNoDispatcher
	}

	// Reset stats for new analysis
	o.stats = Stats{
		StartTime: time.Now(),
	}

	// Try to restore state if exists
	if o.HasState() {
		if err := o.LoadState(); err != nil {
			o.logger.Warn().Err(err).Msg("Failed to restore state, starting fresh")
		} else {
			o.logger.Info().Msg("Resumed from previous state")
		}
	}

	// Initialize current task if starting fresh
	if o.currentTask.AgentType == "" {
		o.currentTask = Task{
			AgentType:       "Explorer",
			TaskDescription: query,
			Context: map[string]interface{}{
				"document_path": documentPath,
				"query":         query,
			},
			Depth:        0,
			ChildResults: make(map[string]interface{}),
		}
	}

	// Trampoline loop
	iterations := 0
	for {
		iterations++

		// Safety checks (check depth first for better error messages)
		if o.currentTask.Depth > o.config.MaxRecursionDepth {
			return nil, ErrMaxDepthExceeded
		}

		if iterations > o.config.MaxIterations {
			return nil, ErrMaxIterationsExceeded
		}

		// Track max depth
		if o.currentTask.Depth > o.stats.MaxDepthReached {
			o.stats.MaxDepthReached = o.currentTask.Depth
		}

		o.logger.Info().
			Str("agent", o.currentTask.AgentType).
			Int("depth", o.currentTask.Depth).
			Int("stack_size", len(o.stack)).
			Msg("Dispatching subagent")

		var result *SubagentResult

		// Check cache first
		if cachedResult := o.CheckCache(&o.currentTask); cachedResult != nil {
			o.stats.CacheHits++
			o.logger.Debug().Msg("Using cached result")
			result = &SubagentResult{
				Type:     ResultTypeAnalysis,
				Analysis: cachedResult,
			}
		} else {
			// Dispatch to subagent
			var err error
			result, err = o.dispatcher(ctx, &o.currentTask)
			if err != nil {
				return nil, fmt.Errorf("subagent dispatch failed: %w", err)
			}
			o.stats.TotalSubagentCalls++
		}

		// Process result (both cached and dispatched results)
		if err := o.processResult(ctx, result); err != nil {
			return nil, err
		}

		// Check if we're done
		if len(o.stack) == 0 && result.IsAnalysis() {
			// Clean up state file on completion
			o.ClearState()

			return result.Analysis, nil
		}

		// Save state after each iteration
		if err := o.SaveState(); err != nil {
			o.logger.Warn().Err(err).Msg("Failed to save state")
		}
	}
}

// processResult handles the result from a subagent dispatch
func (o *Orchestrator) processResult(ctx context.Context, result *SubagentResult) error {
	if result.IsContinuation() {
		// CONTINUATION: Push current task to stack, create new task
		o.logger.Debug().
			Str("continuation_agent", result.Continuation.AgentType).
			Str("return_to", result.Continuation.ReturnTo).
			Msg("Continuation requested")

		// Push current task onto stack
		o.stack = append(o.stack, o.currentTask)

		// Create new task from continuation
		returnTo := result.Continuation.ReturnTo
		o.currentTask = Task{
			AgentType:       result.Continuation.AgentType,
			TaskDescription: result.Continuation.Task,
			Context:         result.Continuation.Context,
			Depth:           o.currentTask.Depth + 1,
			ReturnTo:        &returnTo,
			ChildResults:    make(map[string]interface{}),
		}

		return nil // Continue trampolining
	}

	if result.IsAnalysis() {
		// RESULT: Store result and pop stack
		o.logger.Debug().
			Int("token_count", result.Analysis.TokenCount).
			Float64("cost_usd", result.Analysis.CostUSD).
			Msg("Analysis result received")

		// Update stats
		o.stats.TotalTokens += result.Analysis.TokenCount
		o.stats.TotalCostUSD += result.Analysis.CostUSD

		// Store result in cache
		if err := o.StoreCache(&o.currentTask, result.Analysis); err != nil {
			o.logger.Warn().Err(err).Msg("Failed to cache result")
		}

		// If stack is not empty, inject result into parent and continue
		if len(o.stack) > 0 {
			// Store result
			if o.currentTask.ReturnTo != nil {
				o.results[*o.currentTask.ReturnTo] = result.Analysis
			}

			// Pop parent task from stack
			parentTask := o.stack[len(o.stack)-1]
			o.stack = o.stack[:len(o.stack)-1]

			// Inject child results into parent
			if o.currentTask.ReturnTo != nil {
				if parentTask.ChildResults == nil {
					parentTask.ChildResults = make(map[string]interface{})
				}
				parentTask.ChildResults[*o.currentTask.ReturnTo] = result.Analysis
			}

			o.currentTask = parentTask

			o.logger.Debug().
				Int("new_depth", o.currentTask.Depth).
				Int("stack_size", len(o.stack)).
				Msg("Popped stack, continuing with parent task")

			return nil // Continue trampolining
		}

		// Stack is empty and we have a result - we're done
		return nil
	}

	return fmt.Errorf("unknown result type")
}

// PlaceholderDispatcher is a placeholder for the actual subagent dispatcher
// This will be replaced when Claude Code's Agent SDK integration is available
func PlaceholderDispatcher(ctx context.Context, task *Task) (*SubagentResult, error) {
	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	// For now, always return an analysis result (no recursion)
	// In real implementation, Explorer might return ContinuationRequest
	// to spawn Worker subagents for deeper analysis

	return &SubagentResult{
		Type: ResultTypeAnalysis,
		Analysis: &AnalysisResult{
			Type:       "RESULT",
			Content:    fmt.Sprintf("Analysis of '%s' at depth %d", task.TaskDescription, task.Depth),
			Metadata:   task.Context,
			TokenCount: 1000,
			CostUSD:    0.003,
		},
	}, nil
}

// ParseSubagentResponse parses JSON response from subagent
func ParseSubagentResponse(data []byte) (*SubagentResult, error) {
	// Try to determine type by checking for "type" field
	var typeCheck struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, err
	}

	switch typeCheck.Type {
	case "CONTINUATION":
		var cont ContinuationRequest
		if err := json.Unmarshal(data, &cont); err != nil {
			return nil, err
		}
		return &SubagentResult{
			Type:         ResultTypeContinuation,
			Continuation: &cont,
		}, nil

	case "RESULT":
		var result AnalysisResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return &SubagentResult{
			Type:     ResultTypeAnalysis,
			Analysis: &result,
		}, nil

	default:
		return nil, fmt.Errorf("unknown result type: %s", typeCheck.Type)
	}
}
