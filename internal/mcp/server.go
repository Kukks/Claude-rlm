package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kukks/claude-rlm/internal/hash"
	"github.com/kukks/claude-rlm/internal/orchestrator"
	"github.com/kukks/claude-rlm/internal/storage"
	"github.com/rs/zerolog"
)

// Server implements the MCP protocol server
type Server struct {
	orchestrator *orchestrator.Orchestrator
	storage      storage.Backend
	logger       zerolog.Logger
	tools        []Tool
	version      string
}

// NewServer creates a new MCP server
func NewServer(orch *orchestrator.Orchestrator, store storage.Backend, logger zerolog.Logger, version string) *Server {
	s := &Server{
		orchestrator: orch,
		storage:      store,
		logger:       logger,
		version:      version,
	}
	s.tools = s.defineTools()
	return s
}

// RunStdio runs the MCP server on stdio
func (s *Server) RunStdio(ctx context.Context) error {
	s.logger.Info().Msg("RLM MCP server starting on stdio")

	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse request
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to parse request")
			continue
		}

		// Handle request
		response := s.handleRequest(ctx, &req)

		// Send response
		responseJSON, err := json.Marshal(response)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to marshal response")
			continue
		}

		writer.Write(responseJSON)
		writer.WriteByte('\n')
		writer.Flush()
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// handleRequest processes a JSON-RPC request
func (s *Server) handleRequest(ctx context.Context, req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolCall(ctx, req)
	default:
		return NewErrorResponse(req.ID, MethodNotFound, "method not found")
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *Request) *Response {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "rlm",
			Version: s.version,
		},
		Capabilities: ServerCapabilities{
			Tools: ToolsCapability{
				ListChanged: false,
			},
		},
	}

	return NewResponse(req.ID, result)
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(req *Request) *Response {
	result := map[string]interface{}{
		"tools": s.tools,
	}
	return NewResponse(req.ID, result)
}

// handleToolCall handles the tools/call request
func (s *Server) handleToolCall(ctx context.Context, req *Request) *Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, InvalidParams, "invalid parameters")
	}

	var result *ToolResult
	var err error

	switch params.Name {
	case "rlm_analyze":
		result, err = s.handleAnalyze(ctx, params.Arguments)
	case "rlm_check_freshness":
		result, err = s.handleCheckFreshness(ctx, params.Arguments)
	case "rlm_status":
		result, err = s.handleStatus(ctx, params.Arguments)
	case "rlm_search_rag":
		result, err = s.handleSearchRAG(ctx, params.Arguments)
	default:
		return NewErrorResponse(req.ID, MethodNotFound, fmt.Sprintf("unknown tool: %s", params.Name))
	}

	if err != nil {
		return NewResponse(req.ID, NewErrorToolResult(err.Error()))
	}

	return NewResponse(req.ID, result)
}

// handleAnalyze implements the rlm_analyze tool
func (s *Server) handleAnalyze(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	// Extract parameters
	path := "."
	if p, ok := args["path"].(string); ok {
		path = p
	}

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	focus := ""
	if f, ok := args["focus"].(string); ok {
		focus = f
	}

	forceRefresh := false
	if fr, ok := args["force_refresh"].(bool); ok {
		forceRefresh = fr
	}

	// Check staleness
	if !forceRefresh {
		// Load latest analysis to check staleness
		analyses, err := s.storage.GetAll(ctx)
		if err == nil && len(analyses) > 0 {
			// Get most recent analysis for this path
			latest := analyses[len(analyses)-1]
			if latest.Path == path {
				// Check if stale
				staleness, err := hash.CheckStaleness(latest.FileHashes, path, latest.Timestamp)
				if err == nil && !staleness.Stale {
					suggestion := fmt.Sprintf("Previous analysis is still fresh. Use rlm_search_rag to retrieve results, or set force_refresh=true to re-analyze.\nLast analyzed: %s", latest.Timestamp.Format("2006-01-02 15:04:05"))
					return NewToolResult(suggestion), nil
				}
			}
		}
	}

	// Compute file hashes before analysis
	hasher := hash.NewFileHasher()
	fileHashes, err := hasher.ComputeDirectoryHash(path)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to compute file hashes")
		fileHashes = make(map[string]string)
	}

	// Run analysis
	result, err := s.orchestrator.AnalyzeDocument(ctx, path, query)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Store results in RAG
	analysisData := &storage.AnalysisData{
		Query:      query,
		Focus:      focus,
		Timestamp:  time.Now(),
		Result:     map[string]interface{}{"content": result.Content, "metadata": result.Metadata},
		Stats:      s.orchestrator.GetStats(),
		Path:       path,
		FileHashes: fileHashes,
	}

	if err := s.storage.Store(ctx, analysisData); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to store results")
	}

	// Format response
	stats := s.orchestrator.GetStats()
	response := map[string]interface{}{
		"success":       true,
		"result":        result.Content,
		"stats":         stats,
		"rag_location":  fmt.Sprintf(".rlm/analysis_%s.json", analysisData.ID),
		"files_tracked": len(fileHashes),
	}

	responseJSON, _ := json.MarshalIndent(response, "", "  ")
	return NewToolResult(string(responseJSON)), nil
}

// handleCheckFreshness implements the rlm_check_freshness tool
func (s *Server) handleCheckFreshness(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	path := "."
	if p, ok := args["path"].(string); ok {
		path = p
	}

	// Load latest analysis
	analyses, err := s.storage.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load analyses: %w", err)
	}

	if len(analyses) == 0 {
		return NewToolResult("No previous analyses found."), nil
	}

	// Find most recent analysis for this path
	var latest *storage.AnalysisData
	for i := len(analyses) - 1; i >= 0; i-- {
		if analyses[i].Path == path {
			latest = analyses[i]
			break
		}
	}

	if latest == nil {
		return NewToolResult(fmt.Sprintf("No previous analysis found for path: %s", path)), nil
	}

	// Check staleness
	report, err := hash.CheckStaleness(latest.FileHashes, path, latest.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("staleness check failed: %w", err)
	}

	// Format response
	response := map[string]interface{}{
		"fresh":           !report.Stale,
		"last_analysis":   report.LastAnalysis,
		"total_changes":   report.TotalChanges,
		"changed_files":   report.ChangedFiles,
		"new_files":       report.NewFiles,
		"deleted_files":   report.DeletedFiles,
		"recommendation":  report.Recommendation,
	}

	responseJSON, _ := json.MarshalIndent(response, "", "  ")
	return NewToolResult(string(responseJSON)), nil
}

// handleStatus implements the rlm_status tool
func (s *Server) handleStatus(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	stats := s.orchestrator.GetStats()

	// Check if analysis is in progress
	hasState := s.orchestrator.HasState()

	status := map[string]interface{}{
		"analyzing":  hasState,
		"stats":      stats,
		"storage":    s.storage.Name(),
	}

	statusJSON, _ := json.MarshalIndent(status, "", "  ")
	return NewToolResult(string(statusJSON)), nil
}

// handleSearchRAG implements the rlm_search_rag tool
func (s *Server) handleSearchRAG(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	maxResults := 5
	if mr, ok := args["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	// Search
	results, err := s.storage.Search(ctx, query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Format results
	formattedResults := make([]map[string]interface{}, len(results))
	for i, r := range results {
		formattedResults[i] = map[string]interface{}{
			"query":         r.Data.Query,
			"focus":         r.Data.Focus,
			"timestamp":     r.Data.Timestamp.Format("2006-01-02 15:04:05"),
			"score":         r.Score,
			"search_method": r.SearchMethod,
			"result":        r.Data.Result,
		}
	}

	response := map[string]interface{}{
		"results": formattedResults,
		"count":   len(results),
		"backend": s.storage.Name(),
	}

	responseJSON, _ := json.MarshalIndent(response, "", "  ")
	return NewToolResult(string(responseJSON)), nil
}

// Close cleanly shuts down the server
func (s *Server) Close() error {
	if s.storage != nil {
		return s.storage.Close()
	}
	return nil
}
