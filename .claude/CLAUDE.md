# RLM - Recursive Language Models for Claude

## Project Overview

RLM implements the Recursive Language Models pattern for analyzing documents beyond context window limits using a trampoline-based continuation approach.

## Core Innovation

**Trampoline Pattern:** Bypasses Claude Code's "subagents can't spawn subagents" limitation by having the main session act as a trampoline, bouncing control between depth levels based on continuation requests from subagents.

## Architecture

```
Main Orchestrator (Go)
    ↓ dispatches
Explorer/Worker Subagent
    ↓ returns
ContinuationRequest OR Result
    ↓ orchestrator decides
Continue (push to stack) OR Return (pop stack)
```

## Key Components

1. **Orchestrator** (`internal/orchestrator/orchestrator.go`): Manages trampoline loop, state persistence
2. **MCP Server** (`internal/mcp/server.go`): Exposes RLM as MCP tools
3. **Storage Backend** (`internal/storage/`): BM25-based document indexing
4. **State Persistence**: `.rlm_state.json` for resume capability
5. **Response Caching**: Built-in cache with TTL

## Installation

```bash
# Download latest release and run install
rlm install

# Or manually add to ~/.claude.json:
{
  "mcpServers": {
    "rlm": {
      "command": "path/to/rlm",
      "args": ["mcp"]
    }
  }
}
```

## Usage

### CLI Commands
```bash
# Check status
rlm status

# Direct analysis (for testing)
rlm analyze path/to/document "Your analysis query"

# Update to latest version
rlm update

# Run MCP server (called by Claude)
rlm mcp
```

## Configuration

Create `~/.claude-rlm/config.yaml`:
```yaml
orchestrator:
  max_recursion_depth: 10
  max_iterations: 1000
  cache_enabled: true
  cache_ttl_hours: 24

storage:
  rag_dir: .rlm

logging:
  level: info
  format: text

updater:
  enabled: true
```

## Project Structure
```
claude-rlm/
├── cmd/
│   └── rlm/
│       └── main.go              # CLI entry point
├── internal/
│   ├── config/                  # Configuration loading
│   ├── hash/                    # File hashing & staleness
│   ├── mcp/                     # MCP server implementation
│   │   ├── protocol.go
│   │   ├── server.go
│   │   └── tools.go
│   ├── orchestrator/            # Trampoline orchestrator
│   │   ├── cache.go
│   │   ├── orchestrator.go
│   │   ├── state.go
│   │   └── task.go
│   ├── storage/                 # Document storage & BM25
│   │   ├── backend.go
│   │   ├── bm25.go
│   │   └── models.go
│   └── updater/                 # Self-update functionality
├── go.mod
├── go.sum
└── README.md
```

## Trampoline Pattern Details

The orchestrator maintains a **continuation stack** and executes a loop:

1. Dispatch current task to subagent
2. Receive result (either ContinuationRequest or AnalysisResult)
3. If ContinuationRequest:
   - Push current task to stack
   - Create new task from continuation
   - Loop (trampoline!)
4. If AnalysisResult and stack not empty:
   - Store result
   - Pop parent task from stack
   - Inject result into parent
   - Loop
5. If AnalysisResult and stack empty:
   - Done! Return final result

This pattern allows unlimited recursion depth while respecting the constraint that subagents cannot directly spawn other subagents.

## State Persistence

State is saved after every subagent call, containing:
- Current task
- Stack of parent tasks
- Results collected so far
- Statistics (costs, depths, etc.)

This enables:
- Resume after interruption
- Crash recovery
- Multi-session analysis for very large documents

## Research References

- **RLM Paper**: [arxiv.org/abs/2512.24601](https://arxiv.org/abs/2512.24601)
- **ReDel Framework**: [arxiv.org/abs/2408.02248](https://arxiv.org/abs/2408.02248)
- **Official RLM Implementation**: [github.com/alexzhang13/rlm](https://github.com/alexzhang13/rlm)

## Key Findings from RLM Paper

- **158% improvement** on code analysis tasks vs base model
- **35% cost reduction** (median) vs direct context loading
- Models spontaneously develop verification loops without training
- Peek-before-processing pattern emerges naturally
- Regex filtering reduces tokens by 70-90%

## Development

### Building
```bash
go build -o rlm ./cmd/rlm
```

### Testing
```bash
go test ./...

# Test specific package
go test ./internal/orchestrator/...
```

### Releasing
Releases are automated via GitHub Actions when a new tag is pushed.

## License

MIT License - See LICENSE file for details
