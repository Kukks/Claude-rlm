package mcp

// defineTools returns the list of MCP tools provided by this server
func (s *Server) defineTools() []Tool {
	return []Tool{
		{
			Name:        "rlm_analyze",
			Description: "Analyze code, documentation, or any files using Recursive Language Models (RLM) pattern. Uses intelligent decomposition and trampoline-based recursion to handle documents beyond context limits. Results are stored in local RAG for instant retrieval.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to file or directory to analyze (default: current directory)",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Analysis objective or question (required)",
					},
					"focus": map[string]interface{}{
						"type":        "string",
						"description": "Analysis domain: 'security', 'architecture', 'performance', 'testing', 'documentation', etc.",
					},
					"force_refresh": map[string]interface{}{
						"type":        "boolean",
						"description": "Force re-analysis even if cache is fresh (default: false)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "rlm_check_freshness",
			Description: "Check if previous analysis results are still current or if files have changed. Uses SHA256 file hashing to detect modifications, additions, or deletions since last analysis.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path that was previously analyzed (default: current directory)",
					},
				},
			},
		},
		{
			Name:        "rlm_status",
			Description: "Check status of current or recent RLM analysis. Shows progress, depth, stack size, and resource usage statistics.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "rlm_search_rag",
			Description: "Search previously analyzed content (RAG - Retrieval Augmented Generation). Uses semantic search with Qdrant when available, falls back to keyword search. Results include relevance scores and staleness warnings.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (required)",
					},
					"max_results": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return (default: 5)",
						"minimum":     1,
						"maximum":     50,
					},
				},
				"required": []string{"query"},
			},
		},
	}
}
