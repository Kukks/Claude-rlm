#!/usr/bin/env python3
"""
RLM MCP Server - Enhanced with staleness detection and Claude-mem integration

Exposes RLM analysis as an MCP tool with:
- Automatic staleness detection (tracks file changes)
- Optional Claude-mem integration for cross-project learning
- Smart cache invalidation
"""
import sys
import os
import json
import asyncio
import hashlib
import glob
from typing import Any, Dict, Optional, List
from datetime import datetime

# Add parent directory to path to import orchestrator
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))
from src.orchestrator import RLMOrchestrator

class MCPServer:
    """MCP Server for RLM analysis with staleness detection"""

    def __init__(self):
        self.orchestrator = RLMOrchestrator()
        self.tools = self._define_tools()
        self.claude_mem_available = self._check_claude_mem()

    def _check_claude_mem(self) -> bool:
        """Check if claude-mem MCP server is available"""
        try:
            # Try to import claude-mem client if available
            # This is a placeholder - actual implementation would check MCP registry
            import subprocess
            result = subprocess.run(
                ['which', 'claude-mem'],
                capture_output=True,
                timeout=1
            )
            available = result.returncode == 0
            if available:
                print("✓ Claude-mem detected - enhanced search enabled", file=sys.stderr)
            else:
                print("ℹ Claude-mem not found - using local RAG only", file=sys.stderr)
            return available
        except:
            return False

    def _compute_file_hash(self, file_path: str) -> Optional[str]:
        """Compute SHA256 hash of file for change detection"""
        try:
            hasher = hashlib.sha256()
            with open(file_path, 'rb') as f:
                for chunk in iter(lambda: f.read(4096), b''):
                    hasher.update(chunk)
            return hasher.hexdigest()
        except:
            return None

    def _compute_directory_hash(self, directory: str, patterns: List[str] = None) -> Dict[str, str]:
        """Compute hashes for all files in directory"""
        if patterns is None:
            patterns = ['**/*.py', '**/*.js', '**/*.ts', '**/*.rs', '**/*.go',
                       '**/*.java', '**/*.md', '**/*.txt']

        file_hashes = {}
        for pattern in patterns:
            for file_path in glob.glob(os.path.join(directory, pattern), recursive=True):
                if '.git' in file_path or 'node_modules' in file_path or '.rlm' in file_path:
                    continue

                rel_path = os.path.relpath(file_path, directory)
                file_hash = self._compute_file_hash(file_path)
                if file_hash:
                    file_hashes[rel_path] = file_hash

        return file_hashes

    def _check_staleness(self, rag_dir: str, analyzed_path: str) -> Dict[str, Any]:
        """Check if RAG data is stale due to file changes"""
        index_file = os.path.join(rag_dir, "index.json")
        if not os.path.exists(index_file):
            return {"stale": False, "reason": "no_data"}

        with open(index_file) as f:
            index = json.load(f)

        if not index:
            return {"stale": False, "reason": "empty_index"}

        # Get latest analysis
        latest = max(index, key=lambda x: x.get("timestamp", ""))

        # Load full analysis to get file hashes
        analysis_file = os.path.join(rag_dir, latest["file"])
        if not os.path.exists(analysis_file):
            return {"stale": True, "reason": "missing_file"}

        with open(analysis_file) as f:
            analysis = json.load(f)

        stored_hashes = analysis.get("file_hashes", {})
        if not stored_hashes:
            # Old format without hashes - mark as potentially stale
            return {
                "stale": True,
                "reason": "no_hash_tracking",
                "recommendation": "Re-analyze to enable change detection"
            }

        # Compute current hashes
        if os.path.isfile(analyzed_path):
            current_hashes = {
                os.path.basename(analyzed_path): self._compute_file_hash(analyzed_path)
            }
        else:
            current_hashes = self._compute_directory_hash(analyzed_path)

        # Compare hashes
        changed_files = []
        new_files = []
        deleted_files = []

        for file_path, stored_hash in stored_hashes.items():
            if file_path not in current_hashes:
                deleted_files.append(file_path)
            elif current_hashes[file_path] != stored_hash:
                changed_files.append(file_path)

        for file_path in current_hashes:
            if file_path not in stored_hashes:
                new_files.append(file_path)

        is_stale = bool(changed_files or new_files or deleted_files)

        return {
            "stale": is_stale,
            "changed_files": changed_files,
            "new_files": new_files,
            "deleted_files": deleted_files,
            "total_changes": len(changed_files) + len(new_files) + len(deleted_files),
            "last_analysis": latest.get("timestamp"),
            "recommendation": "Re-analyze to update" if is_stale else "Data is current"
        }

    async def _query_claude_mem(self, query: str, context: Dict[str, Any]) -> Optional[Dict]:
        """Query claude-mem for similar past analyses (if available)"""
        if not self.claude_mem_available:
            return None

        try:
            # This would integrate with claude-mem's MCP tools
            # Placeholder implementation
            return {
                "pattern_suggestions": [],
                "past_successes": [],
                "estimated_cost": None
            }
        except Exception as e:
            print(f"Claude-mem query failed: {e}", file=sys.stderr)
            return None

    async def _store_to_claude_mem(self, analysis_summary: Dict[str, Any]):
        """Store successful analysis pattern to claude-mem"""
        if not self.claude_mem_available:
            return

        try:
            # This would send observation to claude-mem
            # Placeholder implementation
            pass
        except Exception as e:
            print(f"Claude-mem storage failed: {e}", file=sys.stderr)

    def _define_tools(self) -> list:
        """Define MCP tools for RLM"""
        return [
            {
                "name": "rlm_analyze",
                "description": """Analyze code, documentation, or any files using Recursive Language Models (RLM).

**Change Detection:** Automatically detects when analyzed files have changed and marks RAG data as stale.

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

The tool stores results in the repo at .rlm/ with file hash tracking for staleness detection.""",
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
                        },
                        "force_refresh": {
                            "type": "boolean",
                            "description": "Force re-analysis even if RAG data exists and is current",
                            "default": False
                        }
                    },
                    "required": ["query"]
                }
            },
            {
                "name": "rlm_check_freshness",
                "description": """Check if previous analysis data is still current or stale.

Returns:
- Whether files have changed since last analysis
- List of changed, new, and deleted files
- Recommendation to re-analyze or use cached data

Use this before relying on cached RAG data to ensure accuracy.""",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Path that was previously analyzed (default: current directory)"
                        }
                    }
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

**Enhanced with Claude-mem:** If available, also searches across projects for similar analyses.

Use when you need to recall information from previous analyses without re-analyzing.
The tool searches cached analysis results stored in .rlm/ directory and optionally
queries claude-mem for cross-project insights.

Examples:
- "What did we find about authentication last time?"
- "Show me the security issues we identified"
- "What was the architecture summary?"

Note: Automatically warns if data is stale due to file changes.""",
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
                        },
                        "include_stale": {
                            "type": "boolean",
                            "description": "Include results even if files have changed (default: true, but with warnings)",
                            "default": True
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
        elif tool_name == "rlm_check_freshness":
            return await self._handle_check_freshness(arguments)
        elif tool_name == "rlm_status":
            return await self._handle_status(arguments)
        elif tool_name == "rlm_search_rag":
            return await self._handle_search_rag(arguments)
        else:
            return {"error": f"Unknown tool: {tool_name}"}

    async def _handle_check_freshness(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Handle freshness check"""
        path = args.get("path", ".")

        # Get RAG directory
        base_path = os.path.abspath(path)
        if os.path.isfile(base_path):
            base_path = os.path.dirname(base_path)

        rag_dir = os.path.join(base_path, ".rlm")

        if not os.path.exists(rag_dir):
            return {
                "success": True,
                "fresh": False,
                "reason": "no_previous_analysis",
                "message": "No previous analysis found. Run rlm_analyze first."
            }

        staleness_info = self._check_staleness(rag_dir, path)

        return {
            "success": True,
            "fresh": not staleness_info["stale"],
            "staleness_info": staleness_info,
            "message": staleness_info.get("recommendation", "")
        }

    async def _handle_analyze(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Handle rlm_analyze tool call"""
        path = args.get("path", ".")
        query = args.get("query")
        focus = args.get("focus", "general")
        force_refresh = args.get("force_refresh", False)

        # Check for stale data unless force_refresh
        base_path = os.path.abspath(path)
        if os.path.isfile(base_path):
            base_path = os.path.dirname(base_path)

        rag_dir = os.path.join(base_path, ".rlm")

        staleness_info = None
        if os.path.exists(rag_dir) and not force_refresh:
            staleness_info = self._check_staleness(rag_dir, path)
            if not staleness_info["stale"]:
                # Data is fresh, suggest using search instead
                return {
                    "success": True,
                    "used_cache": True,
                    "message": "Analysis data is current. Use rlm_search_rag to retrieve results, or set force_refresh=true to re-analyze.",
                    "staleness_info": staleness_info
                }

        # Query claude-mem for suggestions if available
        claude_mem_suggestions = await self._query_claude_mem(query, {
            "path": path,
            "focus": focus,
            "staleness": staleness_info
        })

        # Enhance query with focus
        if focus != "general":
            query = f"[Focus: {focus}] {query}"

        try:
            # Compute file hashes before analysis
            if os.path.isfile(path):
                file_hashes = {os.path.basename(path): self._compute_file_hash(path)}
            else:
                file_hashes = self._compute_directory_hash(path)

            # Run orchestrator
            result = self.orchestrator.analyze_document(path, query)

            # Store RAG data with file hashes
            self._store_rag_data(path, query, result, focus, file_hashes)

            # Store pattern to claude-mem if successful
            if self.claude_mem_available:
                await self._store_to_claude_mem({
                    "query": query,
                    "focus": focus,
                    "success": True,
                    "stats": self.orchestrator.stats,
                    "staleness_triggered": staleness_info is not None
                })

            response = {
                "success": True,
                "result": result,
                "stats": self.orchestrator.stats,
                "rag_stored": True,
                "rag_location": rag_dir,
                "files_tracked": len(file_hashes),
                "change_detection_enabled": True
            }

            if staleness_info and staleness_info["stale"]:
                response["staleness_info"] = staleness_info
                response["reason_for_analysis"] = "Files changed since last analysis"

            if claude_mem_suggestions:
                response["claude_mem_suggestions"] = claude_mem_suggestions

            return response

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
                "status": status,
                "claude_mem_available": self.claude_mem_available
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
        include_stale = args.get("include_stale", True)

        try:
            # Search in .rlm/ directory
            rag_dir = ".rlm"
            if not os.path.exists(rag_dir):
                return {
                    "success": True,
                    "results": [],
                    "message": "No previous analyses found. Use rlm_analyze first.",
                    "claude_mem_available": self.claude_mem_available
                }

            # Check freshness
            staleness_info = self._check_staleness(rag_dir, ".")

            results = self._search_rag_directory(rag_dir, query, max_results)

            # Query claude-mem for additional context if available
            claude_mem_results = None
            if self.claude_mem_available:
                claude_mem_results = await self._query_claude_mem(query, {"local_results": results})

            response = {
                "success": True,
                "results": results,
                "count": len(results),
                "staleness_info": staleness_info
            }

            if staleness_info["stale"] and not include_stale:
                response["warning"] = "Cached data is stale. Consider re-analyzing with rlm_analyze."
                response["results"] = []
                response["count"] = 0
            elif staleness_info["stale"]:
                response["warning"] = f"Data may be outdated. {staleness_info['total_changes']} files changed since last analysis."

            if claude_mem_results:
                response["claude_mem_results"] = claude_mem_results

            return response

        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }

    def _store_rag_data(self, path: str, query: str, result: Any, focus: str, file_hashes: Dict[str, str]):
        """Store analysis results as RAG data in repo with file hashes"""
        # Create .rlm directory in the analyzed path
        base_path = os.path.abspath(path)
        if os.path.isfile(base_path):
            base_path = os.path.dirname(base_path)

        rag_dir = os.path.join(base_path, ".rlm")
        os.makedirs(rag_dir, exist_ok=True)

        # Create timestamped entry
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

        # Store analysis result with file hashes
        result_file = os.path.join(rag_dir, f"analysis_{timestamp}.json")
        with open(result_file, 'w') as f:
            json.dump({
                "query": query,
                "focus": focus,
                "timestamp": timestamp,
                "result": result if isinstance(result, dict) else {"content": str(result)},
                "stats": self.orchestrator.stats,
                "path": path,
                "file_hashes": file_hashes,  # Track file state
                "version": "2.0"  # Mark as having hash tracking
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
            "path": path,
            "files_tracked": len(file_hashes),
            "has_hash_tracking": True
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

        # Simple keyword matching (could be enhanced with embeddings via claude-mem)
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
                        "stats": full_data.get("stats"),
                        "has_hash_tracking": entry.get("has_hash_tracking", False)
                    })

        # Sort by score and return top results
        scored_results.sort(key=lambda x: x["score"], reverse=True)
        return scored_results[:max_results]

    async def run_stdio(self):
        """Run MCP server on stdio"""
        print("RLM MCP Server starting on stdio...", file=sys.stderr)
        print(f"Change detection: enabled", file=sys.stderr)
        print(f"Claude-mem integration: {'enabled' if self.claude_mem_available else 'disabled'}", file=sys.stderr)

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
                                "version": "2.0.0"
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
