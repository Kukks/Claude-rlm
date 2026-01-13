package orchestrator

import "time"

// Task represents work to be performed by a subagent
type Task struct {
	AgentType       string                 `json:"agent_type"`
	TaskDescription string                 `json:"task_description"`
	Context         map[string]interface{} `json:"context"`
	Depth           int                    `json:"depth"`
	ReturnTo        *string                `json:"return_to,omitempty"`
	ChildResults    map[string]interface{} `json:"child_results,omitempty"`
}

// ContinuationRequest signals that recursion is needed
type ContinuationRequest struct {
	Type      string                 `json:"type"` // Always "CONTINUATION"
	AgentType string                 `json:"agent_type"`
	Task      string                 `json:"task"`
	Context   map[string]interface{} `json:"context"`
	ReturnTo  string                 `json:"return_to"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AnalysisResult is the completed work from a subagent
type AnalysisResult struct {
	Type       string                 `json:"type"` // Always "RESULT"
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	TokenCount int                    `json:"token_count"`
	CostUSD    float64                `json:"cost_usd"`
}

// Stats tracks analysis metrics
type Stats struct {
	TotalSubagentCalls int       `json:"total_subagent_calls"`
	TotalTokens        int       `json:"total_tokens"`
	TotalCostUSD       float64   `json:"total_cost_usd"`
	MaxDepthReached    int       `json:"max_depth_reached"`
	CacheHits          int       `json:"cache_hits"`
	StartTime          time.Time `json:"start_time"`
}

// State represents the orchestrator state for persistence
type State struct {
	Stack       []Task                 `json:"stack"`
	CurrentTask Task                   `json:"current_task"`
	Results     map[string]interface{} `json:"results"`
	Stats       Stats                  `json:"stats"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ResultType represents the type of result from subagent dispatch
type ResultType int

const (
	ResultTypeContinuation ResultType = iota
	ResultTypeAnalysis
	ResultTypeUnknown
)

// SubagentResult is a union type for subagent responses
type SubagentResult struct {
	Type         ResultType
	Continuation *ContinuationRequest
	Analysis     *AnalysisResult
}

// IsContinuation checks if the result is a continuation request
func (r *SubagentResult) IsContinuation() bool {
	return r.Type == ResultTypeContinuation && r.Continuation != nil
}

// IsAnalysis checks if the result is an analysis result
func (r *SubagentResult) IsAnalysis() bool {
	return r.Type == ResultTypeAnalysis && r.Analysis != nil
}
