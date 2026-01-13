#!/usr/bin/env python3
"""
RLM MCP Server - Model Context Protocol server for Recursive Language Models

Exposes RLM analysis as an MCP tool that can be automatically triggered
by natural language queries about code, documentation, and repositories.
"""
import sys
import os
import json
import asyncio
from typing import Any, Dict, Optional

# Add parent directory to path to import orchestrator
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))
from src.orchestrator import RLMOrchestrator

# MCP Protocol implementation
class MCPServer:
    """MCP Server for RLM analysis"""

    def __init__(self):
        self.orchestrator = RLMOrchestrator()
        self.tools = self._define_tools()

    def _define_tools(self) -> list:
        """Define MCP tools for RLM"""
        return [
            {
                "name": "rlm_analyze",
                "description": """Analyze code, documentation, or any files using Recursive Language Models (RLM).

Use this tool when asked to:
- Understand a codebase or repository structure
- Find specific patterns, functions, or vulnerabilities across many files
- Analyze documentation for completeness or accuracy
- Review code for security issues, bugs, or best practices
- Explain how a large system works
- Compare implementations across multiple files
- Find all occurrences of a pattern with context

This tool can handle codebases and documents far beyond the context window by
intelligently decomposing the analysis into smaller tasks.

Examples of when to use:
- "How does authentication work in this repo?"
- "Find all SQL queries and check for injection vulnerabilities"
- "What are the main components of this system?"
- "Analyze the API documentation for missing endpoints"
- "Explain the data flow through the application"

The tool stores results in the repo at .rlm/ for future reference.""",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Path to file or directory to analyze (default: current directory)"
                        },
                        "query": {
                            "type": "string",
                            "description": "What to analyze or find (e.g., 'Find security vulnerabilities', 'Explain authentication flow')"
                        },
                        "focus": {
                            "type": "string",
                            "description": "Optional: Specific focus area (e.g., 'security', 'performance', 'documentation')",
                            "enum": ["security", "architecture", "performance", "documentation", "testing", "general"]
                        }
                    },
                    "required": ["query"]
                }
            },
            {
                "name": "rlm_status",
                "description": "Check status of current or recent RLM analysis. Shows progress, costs, and recursion depth.",
                "inputSchema": {
                    "type": "object",
                    "properties": {}
                }
            },
            {
                "name": "rlm_search_rag",
                "description": """Search previously analyzed content (RAG - Retrieval Augmented Generation).

Use when you need to recall information from previous analyses without re-analyzing.
The tool searches cached analysis results stored in .rlm/ directory.

Examples:
- "What did we find about authentication last time?"
- "Show me the security issues we identified"
- "What was the architecture summary?"
""",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "What to search for in previous analyses"
                        },
                        "max_results": {
                            "type": "integer",
                            "description": "Maximum number of results to return (default: 5)",
                            "default": 5
                        }
                    },
                    "required": ["query"]
                }
            }
        ]

    async def handle_tool_call(self, tool_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """Handle MCP tool calls"""

        if tool_name == "rlm_analyze":
            return await self._handle_analyze(arguments)
        elif tool_name == "rlm_status":
            return await self._handle_status(arguments)
        elif tool_name == "rlm_search_rag":
            return await self._handle_search_rag(arguments)
        else:
            return {"error": f"Unknown tool: {tool_name}"}

    async def _handle_analyze(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Handle rlm_analyze tool call"""
        path = args.get("path", ".")
        query = args.get("query")
        focus = args.get("focus", "general")

        # Enhance query with focus
        if focus != "general":
            query = f"[Focus: {focus}] {query}"

        try:
            # Run orchestrator
            result = self.orchestrator.analyze_document(path, query)

            # Store RAG data
            self._store_rag_data(path, query, result, focus)

            return {
                "success": True,
                "result": result,
                "stats": self.orchestrator.stats,
                "rag_stored": True,
                "rag_location": os.path.join(os.path.abspath(path), ".rlm")
            }
        except Exception as e:
            return {
                "success": False,
                "error": str(e),
                "type": type(e).__name__
            }

    async def _handle_status(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Handle rlm_status tool call"""
        try:
            status = self.orchestrator.get_status()
            return {
                "success": True,
                "status": status
            }
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }

    async def _handle_search_rag(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Handle rlm_search_rag tool call"""
        query = args.get("query")
        max_results = args.get("max_results", 5)

        try:
            # Search in .rlm/ directory
            rag_dir = ".rlm"
            if not os.path.exists(rag_dir):
                return {
                    "success": True,
                    "results": [],
                    "message": "No previous analyses found. Use rlm_analyze first."
                }

            results = self._search_rag_directory(rag_dir, query, max_results)

            return {
                "success": True,
                "results": results,
                "count": len(results)
            }
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }

    def _store_rag_data(self, path: str, query: str, result: Any, focus: str):
        """Store analysis results as RAG data in repo"""
        # Create .rlm directory in the analyzed path
        base_path = os.path.abspath(path)
        if os.path.isfile(base_path):
            base_path = os.path.dirname(base_path)

        rag_dir = os.path.join(base_path, ".rlm")
        os.makedirs(rag_dir, exist_ok=True)

        # Create timestamped entry
        from datetime import datetime
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

        # Store analysis result
        result_file = os.path.join(rag_dir, f"analysis_{timestamp}.json")
        with open(result_file, 'w') as f:
            json.dump({
                "query": query,
                "focus": focus,
                "timestamp": timestamp,
                "result": result if isinstance(result, dict) else {"content": str(result)},
                "stats": self.orchestrator.stats,
                "path": path
            }, f, indent=2)

        # Update index
        index_file = os.path.join(rag_dir, "index.json")
        index = []
        if os.path.exists(index_file):
            with open(index_file) as f:
                index = json.load(f)

        index.append({
            "timestamp": timestamp,
            "query": query,
            "focus": focus,
            "file": f"analysis_{timestamp}.json",
            "path": path
        })

        with open(index_file, 'w') as f:
            json.dump(index, f, indent=2)

    def _search_rag_directory(self, rag_dir: str, query: str, max_results: int) -> list:
        """Search RAG data for relevant past analyses"""
        results = []

        index_file = os.path.join(rag_dir, "index.json")
        if not os.path.exists(index_file):
            return results

        with open(index_file) as f:
            index = json.load(f)

        # Simple keyword matching (could be enhanced with embeddings)
        query_lower = query.lower()
        query_words = set(query_lower.split())

        scored_results = []
        for entry in index:
            entry_query = entry.get("query", "").lower()
            entry_focus = entry.get("focus", "").lower()

            # Calculate relevance score
            score = 0
            if query_lower in entry_query:
                score += 10

            entry_words = set(entry_query.split())
            common_words = query_words & entry_words
            score += len(common_words) * 2

            if query_lower in entry_focus:
                score += 5

            if score > 0:
                # Load the actual result
                result_file = os.path.join(rag_dir, entry["file"])
                if os.path.exists(result_file):
                    with open(result_file) as f:
                        full_data = json.load(f)

                    scored_results.append({
                        "score": score,
                        "timestamp": entry["timestamp"],
                        "query": entry["query"],
                        "focus": entry["focus"],
                        "result": full_data.get("result"),
                        "stats": full_data.get("stats")
                    })

        # Sort by score and return top results
        scored_results.sort(key=lambda x: x["score"], reverse=True)
        return scored_results[:max_results]

    async def run_stdio(self):
        """Run MCP server on stdio"""
        print("RLM MCP Server starting on stdio...", file=sys.stderr)

        while True:
            try:
                # Read JSON-RPC message from stdin
                line = sys.stdin.readline()
                if not line:
                    break

                message = json.loads(line)

                # Handle different message types
                if message.get("method") == "tools/list":
                    response = {
                        "jsonrpc": "2.0",
                        "id": message.get("id"),
                        "result": {
                            "tools": self.tools
                        }
                    }
                    print(json.dumps(response), flush=True)

                elif message.get("method") == "tools/call":
                    tool_name = message["params"]["name"]
                    arguments = message["params"].get("arguments", {})

                    result = await self.handle_tool_call(tool_name, arguments)

                    response = {
                        "jsonrpc": "2.0",
                        "id": message.get("id"),
                        "result": {
                            "content": [
                                {
                                    "type": "text",
                                    "text": json.dumps(result, indent=2)
                                }
                            ]
                        }
                    }
                    print(json.dumps(response), flush=True)

                elif message.get("method") == "initialize":
                    response = {
                        "jsonrpc": "2.0",
                        "id": message.get("id"),
                        "result": {
                            "protocolVersion": "2024-11-05",
                            "capabilities": {
                                "tools": {}
                            },
                            "serverInfo": {
                                "name": "rlm-server",
                                "version": "1.0.0"
                            }
                        }
                    }
                    print(json.dumps(response), flush=True)

            except json.JSONDecodeError:
                continue
            except Exception as e:
                print(f"Error: {e}", file=sys.stderr)
                continue

async def main():
    server = MCPServer()
    await server.run_stdio()

if __name__ == "__main__":
    asyncio.run(main())
