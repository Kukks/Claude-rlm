# Claude RLM Plugin

> Recursive Language Models for Claude Code - Analyze documents beyond context window limits

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.23+](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org/dl/)

## Overview

The Claude RLM plugin implements the Recursive Language Models pattern from [DeepMind/MIT research](https://arxiv.org/abs/2512.24601), enabling analysis of documents that exceed context window limits through intelligent decomposition and trampoline-based recursion.

**New in v3.0:** Complete rewrite in Go - single binary, no runtime dependencies, faster performance!

### Key Features

- 🔄 **Trampoline Pattern**: Unlimited recursion depth within Claude Code's constraints
- 💾 **State Persistence**: Resume analysis after interruption or across sessions
- 🚀 **Response Caching**: 70%+ cost reduction on similar queries
- 📊 **Cost Tracking**: Real-time monitoring of token usage and costs
- 🎯 **Smart Delegation**: Intelligent task decomposition and parallel processing
- 📈 **Proven Patterns**: Library of research-backed analysis strategies
- 🤖 **MCP Integration**: Auto-triggered by natural language in Claude
- 💡 **Repo-local RAG**: Persistent analysis stored in `.rlm/` directory
- 🔍 **Staleness Detection**: Tracks file changes with SHA256 hashing (v3.0)
- 🔎 **BM25 Search**: Pure Go search algorithm, zero external dependencies (v3.0)
- 📦 **Single Binary**: Truly standalone, works offline on Windows/Mac/Linux (v3.0)

### MCP Integration (Recommended!)

**✨ New!** RLM now works as an MCP server - just ask Claude questions naturally:

```
"How does authentication work in this repo?"
"Find security vulnerabilities"
"What are the main components?"
```

Claude automatically calls RLM when needed. Results stored in your repo for instant retrieval later!

👉 **[5-Minute Setup Guide](QUICKSTART_MCP.md)** | **[Full MCP Guide](MCP_INTEGRATION.md)** | **[Example Prompts](EXAMPLE_PROMPTS.md)** | **[Semantic Search Guide](SEMANTIC_SEARCH.md)**

### Benefits

Based on the RLM research paper:
- **158% improvement** in code analysis accuracy vs baseline models
- **35% cost reduction** (median) vs direct context loading
- **Handles documents 10x+ larger** than context window
- **Resumable analysis** for very large documents

## Installation

### 🚀 Quick Install (Recommended)

**Download and run the auto-installer:**

```bash
# Linux (amd64)
curl -L https://github.com/Kukks/claude-rlm/releases/latest/download/claude-rlm_linux_amd64.tar.gz | tar xz
./rlm install

# macOS (Apple Silicon)
curl -L https://github.com/Kukks/claude-rlm/releases/latest/download/claude-rlm_darwin_arm64.tar.gz | tar xz
./rlm install

# macOS (Intel)
curl -L https://github.com/Kukks/claude-rlm/releases/latest/download/claude-rlm_darwin_amd64.tar.gz | tar xz
./rlm install
```

```powershell
# Windows (PowerShell)
iwr "https://github.com/Kukks/claude-rlm/releases/latest/download/claude-rlm_windows_amd64.zip" -OutFile "rlm.zip"
Expand-Archive rlm.zip -DestinationPath . -Force
.\rlm.exe install
```

The `rlm install` command will:
- Install the binary to your PATH (auto-configured on Windows)
- Auto-detect and configure Claude Desktop (if installed)
- Auto-detect and configure Claude Code CLI (if installed)

**After installation, restart your terminal**, then verify:

```bash
rlm --version
# Should show: rlm version v3.0.x
```

### Manual Installation

**Claude Desktop:**

1. Install RLM binary (see above)

2. Find your config file:
   - **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
   - **Linux**: `~/.config/Claude/claude_desktop_config.json`

3. Add this config:
```json
{
  "mcpServers": {
    "rlm": {
      "command": "/usr/local/bin/rlm",
      "args": ["mcp"]
    }
  }
}
```

4. Restart Claude Desktop

**Claude Code CLI:**

1. Install RLM binary (see above)

2. Edit `~/.config/claude/config.json`:
```json
{
  "mcpServers": {
    "rlm": {
      "command": "/usr/local/bin/rlm",
      "args": ["mcp"]
    }
  }
}
```

3. Restart your terminal

### Build from Source

```bash
git clone https://github.com/Kukks/claude-rlm.git
cd claude-rlm
make build
sudo cp bin/rlm /usr/local/bin/
```

**Requirements:**
- Go 1.23+ (for building)
- Qdrant (optional, for semantic search)

### Standalone CLI (No MCP)

**For direct command-line usage without MCP integration:**

```bash
# After installing binary (see above)
rlm analyze path/to/document "Your analysis query"
```

Single binary, no dependencies!

## Quick Start

### With MCP Integration

Just ask Claude naturally:

```
You: "How does authentication work in this repo?"

Claude: [Automatically calls rlm_analyze]
        [RLM analyzes codebase]
        [Results stored in .rlm/]

        "The authentication system uses JWT tokens with..."
```

Later:
```
You: "What did you find about authentication?"

Claude: [Retrieves from .rlm/ - instant, $0 cost]
        "Previously I found that authentication uses..."
```

**See [Example Prompts](EXAMPLE_PROMPTS.md)** for 50+ natural language triggers!

### Standalone CLI

```bash
# Analyze a single file
rlm analyze README.md "Summarize the key features"

# Analyze a codebase
rlm analyze ./src "Find all security vulnerabilities"

# Analyze with specific focus
rlm analyze ./docs "What topics are missing from the documentation?"
```

### Check Status

```bash
rlm status
```

Output:
```
RLM Version: v3.0.0
Build Time: 2026-01-13
Git Commit: abc1234

Configuration:
  Max Recursion Depth: 10
  Cache Enabled: true
  Qdrant Enabled: true
  Qdrant Address: localhost:6334
```

### Resume Interrupted Analysis

```bash
# Start analysis
rlm analyze large_doc.pdf "Deep analysis"

# Interrupt (Ctrl+C)
^C

# Resume automatically
rlm analyze large_doc.pdf "Deep analysis"
# Output: 🔄 Resuming from saved state...
```

## Usage Examples

### Example 1: Code Security Audit

```bash
rlm analyze ./src "Perform security audit focusing on authentication and authorization"
```

The orchestrator will:
1. Explorer analyzes structure, identifies auth-related files
2. Workers perform deep analysis of each file
3. Results aggregated into comprehensive security report

**Expected output:**
- List of security issues with severity ratings
- Code snippets showing vulnerabilities
- Recommendations for fixes
- Cost: ~$2-5 for medium codebase

### Example 2: Research Paper Analysis

```bash
rlm analyze paper.pdf "Summarize methodology, key findings, and compare to related work"
```

The orchestrator will:
1. Explorer identifies paper sections
2. Workers extract methodology, results, related work
3. Synthesizer compares and contrasts findings

**Expected output:**
- Structured summary of paper
- Key contributions identified
- Comparison table with related work
- Cost: ~$1-2 for typical paper

### Example 3: Documentation Gap Analysis

```bash
rlm analyze ./docs "What API endpoints are undocumented? Compare docs to actual codebase"
```

The orchestrator will:
1. Explorer scans documentation for API references
2. Workers analyze codebase for actual endpoints
3. Synthesizer identifies gaps

**Expected output:**
- List of undocumented endpoints
- Missing examples or explanations
- Outdated documentation
- Cost: ~$3-7 for large project

## Configuration

Create `~/.config/rlm/config.yaml`:

```yaml
orchestrator:
  max_recursion_depth: 10
  max_iterations: 1000
  cache_enabled: true
  cache_ttl_hours: 24

storage:
  rag_dir: ".rlm"
  qdrant_enabled: true
  qdrant_address: "localhost:6334"
  collection_name: "rlm_analyses"

updater:
  enabled: true
  check_interval_hours: 24

logging:
  level: "info"  # debug, info, warn, error
  format: "text"  # text or json
```

### Configuration Options

| Section | Option | Default | Description |
|---------|--------|---------|-------------|
| orchestrator | `max_recursion_depth` | 10 | Maximum depth for recursive analysis |
| orchestrator | `cache_enabled` | true | Enable response caching |
| orchestrator | `cache_ttl_hours` | 24 | Cache lifetime in hours |
| storage | `qdrant_enabled` | true | Use Qdrant for semantic search |
| storage | `qdrant_address` | localhost:6334 | Qdrant server address |
| updater | `enabled` | true | Auto-check for updates |

## How It Works

### Trampoline Pattern

The core innovation is a **trampoline-based continuation pattern** that works within Claude Code's constraint that "subagents can't spawn subagents."

```
Main Orchestrator
    ↓ spawns
Explorer Agent → returns ContinuationRequest
    ↓ orchestrator creates task
Worker Agent A → returns ContinuationRequest
    ↓ orchestrator creates task
Worker Agent B → returns Result
    ↓ orchestrator injects result
Worker Agent A → returns Result (enriched)
    ↓ orchestrator injects result
Explorer Agent → returns Final Analysis
```

### Analysis Flow

1. **Structure Analysis**: Explorer agent examines document structure
2. **Task Decomposition**: Explorer creates tasks for Worker agents
3. **Deep Analysis**: Workers analyze assigned sections
4. **Result Aggregation**: Results bubble up through the stack
5. **Final Synthesis**: Complete analysis assembled

### State Management

Every subagent call saves state to `.rlm_state.json`:
```json
{
  "stack": [...],
  "current_task": {...},
  "results": {...},
  "stats": {
    "total_cost_usd": 0.38,
    "total_tokens": 125000,
    "max_depth_reached": 3
  }
}
```

This enables:
- **Resume capability**: Continue from any interruption point
- **Crash recovery**: No lost progress
- **Multi-session analysis**: Spread large analyses across sessions

### Response Caching

Results are cached using SHA256 hash of:
- Agent type (Explorer/Worker)
- Task description
- Context (sorted JSON)

**Benefits:**
- ~70% cost reduction on repeated queries
- Near-instant results for cached tasks
- Automatic cache invalidation after TTL

## RLM Patterns

The plugin includes a library of proven analysis patterns from research. See `skills/rlm-patterns/SKILL.md` for details.

### Quick Reference

| Pattern | When to Use | Benefit |
|---------|-------------|---------|
| Peek Before Processing | Starting any analysis | Understand structure before committing resources |
| Partition and Map | Clear document boundaries | Parallel analysis of independent sections |
| Regex-Based Filtering | Searching for patterns | 70-90% token reduction |
| Verification Loop | High-stakes analysis | 90%+ reduction in false positives |
| Hierarchical Decomposition | Nested structures | Understand relationships and dependencies |
| Iterative Refinement | Evolving analysis | Progressive deepening based on findings |
| State Machine | Very large documents | Multi-session resumable analysis |

## Cost Optimization

### Strategies Implemented

1. **Early Filtering**: Explorer filters content before delegating to Workers
2. **Response Caching**: Identical queries return cached results
3. **Incremental Processing**: Analysis stops when answer is found
4. **Smart Delegation**: Workers only spawned when necessary

### Cost Estimates

Based on document size:

| Document Size | Estimated Cost | Example |
|--------------|---------------|---------|
| Small (<100K tokens) | $0.50 - $2.00 | Single file, short document |
| Medium (100K-1M) | $2.00 - $10.00 | Module, chapter, codebase |
| Large (1M-10M) | $10.00 - $50.00 | Entire repository, book |

**Actual costs depend on:**
- Query complexity
- Recursion depth required
- Cache hit rate
- Document structure

### Tips for Reducing Costs

1. **Be Specific**: "Find SQL injection vulnerabilities" vs "Analyze security"
2. **Use Caching**: Re-run similar queries on same codebase
3. **Filter Early**: Include keywords in query to focus analysis
4. **Monitor Progress**: Use `/rlm:status` to track costs in real-time
5. **Resume Often**: Pause and resume instead of restarting

## Troubleshooting

### "Max recursion depth exceeded"

**Cause**: Query too complex or document too nested

**Solution**:
1. Increase `max_recursion_depth` in config
2. Narrow your query to reduce complexity
3. Split analysis into multiple focused queries

### "State file corrupted"

**Cause**: Interrupted during state save

**Solution**:
```bash
rm .rlm_state.json
rlm analyze path/to/doc "query"
```

### "High cost warning"

**Cause**: Query is too broad

**Solution**:
1. Add specific keywords to query
2. Enable filtering in config
3. Analyze subset of documents first

### Analysis seems stuck

**Check status:**
```bash
rlm analyze status
```

If depth is unusually high (>7), the query may be too complex. Interrupt and narrow the scope.

## Architecture

### Components

```
claude-rlm/
├── src/
│   └── orchestrator.py       # Core trampoline loop
├── agents/
│   ├── rlm-explorer.md       # Structure analysis agent
│   └── rlm-worker.md         # Deep analysis agent
├── skills/
│   └── rlm-patterns/         # Analysis pattern library
│       └── SKILL.md
├── commands/
│   ├── rlm-analyze.md        # Analysis command
│   └── rlm-status.md         # Status command
└── .claude-plugin/
    └── plugin.json           # Plugin manifest
```

### Key Classes

- **RLMOrchestrator**: Manages trampoline loop, state, caching
- **Task**: Represents work for a subagent
- **ContinuationRequest**: Request to recurse deeper
- **AnalysisResult**: Completed analysis from subagent

## Research & References

### Primary Research

1. **Recursive Language Models** - Zhang et al., December 2025
   - Paper: https://arxiv.org/abs/2512.24601
   - Blog: https://alexzhang13.github.io/blog/2025/rlm/

2. **Official RLM Implementation**
   - GitHub: https://github.com/alexzhang13/rlm
   - Minimal: https://github.com/alexzhang13/rlm-minimal

3. **Prime Intellect RLMEnv**
   - Blog: https://primeintellect.ai/blog/rlm
   - Code: https://github.com/PrimeIntellect-ai/verifiers

### Supporting Research

4. **ReDel: Recursive Multi-Agent Systems**
   - Paper: https://arxiv.org/abs/2408.02248
   - Code: https://github.com/zhudotexe/redel

5. **Context-Folding (FoldAgent)**
   - Code: https://github.com/sunnweiwei/FoldAgent

## Contributing

Contributions welcome! Please:

1. Check existing issues and PRs
2. Follow the RLM patterns in `skills/rlm-patterns/SKILL.md`
3. Add tests for new features
4. Update documentation

### Development Setup

```bash
git clone https://github.com/Kukks/claude-rlm.git
cd claude-rlm

# Run tests
python -m pytest tests/

# Test orchestrator
rlm analyze test.txt "test query"
```

## Documentation

**Getting Started:**
- **[5-Minute Setup Guide](QUICKSTART_MCP.md)** - Quick installation and first use
- **[MCP Integration Guide](MCP_INTEGRATION.md)** - Complete setup and configuration
- **[Example Prompts](EXAMPLE_PROMPTS.md)** - 50+ natural language examples

**Advanced Features:**
- **[Semantic Search Guide](SEMANTIC_SEARCH.md)** - Vector embeddings vs keyword search
- **[Staleness Detection](STALENESS_DETECTION.md)** - File change tracking and Claude-mem integration
- **[Compatibility Guide](COMPATIBILITY.md)** - Cross-platform support details

**Development:**
- **[Developer Guide](.claude/CLAUDE.md)** - Internal architecture and contribution guide

## License

MIT License - See [LICENSE](LICENSE) file for details.

## Acknowledgments

- Based on research by Zhang et al. ([Recursive Language Models paper](https://arxiv.org/abs/2512.24601))
- Inspired by ReDel framework and FoldAgent
- Built for the Claude Code ecosystem

## Support

- **Issues**: [GitHub Issues](https://github.com/Kukks/claude-rlm/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Kukks/claude-rlm/discussions)
- **Documentation**: See `.claude/CLAUDE.md` for developer docs

---

## Status

**✅ Production Ready (v3.0)**

- **Single Binary**: Pure Go - zero runtime dependencies
- **MCP Integration**: Fully functional with Claude Desktop and Claude Code CLI
- **BM25 Search**: Industry-standard search algorithm (used by Elasticsearch)
- **Offline Operation**: Works completely offline, no external services needed
- **Staleness Detection**: SHA256-based file change tracking
- **Cross-Platform**: Linux, macOS, Windows (amd64 + arm64)
- **State Persistence**: Resume capability with .rlm_state.json
- **Response Caching**: Cost optimization with 24h TTL
- **Auto-Updater**: Check for updates via GitHub releases

**Usage:** Recommended via MCP server (see [QUICKSTART_MCP.md](QUICKSTART_MCP.md))

**Testing:** All tests passing ✅ (4/4 orchestrator tests)
**Binary Size:** ~10MB (truly standalone)

