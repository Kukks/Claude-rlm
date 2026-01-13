# MCP Integration - Auto-Triggered RAG Analysis

## Overview

The RLM MCP integration allows Claude to **automatically** analyze your codebase when you ask natural language questions. Analysis results are stored in your repo as **persistent RAG data**.

## Features

✅ **Auto-triggered** - Claude calls RLM when you ask about code
✅ **Repo-local RAG** - Results stored in `.rlm/` directory in each repo
✅ **Natural language** - Just ask questions, no commands needed
✅ **Persistent cache** - Previous analyses available instantly
✅ **Cost tracking** - See token usage and costs

## Installation

### 1. Clone RLM Repository

```bash
git clone https://github.com/Kukks/claude-rlm.git
cd claude-rlm
```

### 2. Find Your Claude Config

**macOS:**
```bash
~/Library/Application Support/Claude/claude_desktop_config.json
```

**Windows:**
```bash
%APPDATA%\Claude\claude_desktop_config.json
```

**Linux:**
```bash
~/.config/Claude/claude_desktop_config.json
```

### 3. Add RLM MCP Server

Edit your `claude_desktop_config.json` and add:

```json
{
  "mcpServers": {
    "rlm": {
      "command": "python3",
      "args": [
        "/full/path/to/claude-rlm/mcp_server/rlm_server.py"
      ],
      "env": {}
    }
  }
}
```

**⚠️ Important:** Replace `/full/path/to/claude-rlm` with your actual path!

**Example:**
```json
{
  "mcpServers": {
    "rlm": {
      "command": "python3",
      "args": [
        "/Users/alice/projects/claude-rlm/mcp_server/rlm_server.py"
      ],
      "env": {}
    }
  }
}
```

### 4. Restart Claude Desktop

Completely quit and restart Claude Desktop app.

### 5. Verify Installation

In Claude chat, you should see an MCP indicator showing "rlm" is connected.

## Usage

### Natural Language Triggers

Just ask Claude about your code - RLM will automatically engage!

**Examples that trigger RLM:**

```
"How does authentication work in this repo?"
→ Claude sees this needs codebase analysis, calls rlm_analyze

"Find all SQL queries and check for injection vulnerabilities"
→ RLM analyzes code for security patterns

"What are the main components of this system?"
→ RLM performs architectural analysis

"Explain the data flow through the application"
→ RLM traces data through multiple files
```

### RAG Storage

Results are stored in **each repo** at `.rlm/`:

```
your-project/
├── .rlm/
│   ├── index.json                    # Search index
│   ├── analysis_20260113_140532.json # Stored analysis
│   ├── analysis_20260113_141205.json
│   └── analysis_20260113_142018.json
├── src/
└── ...
```

### Retrieving Past Analyses

Claude automatically searches RAG data when relevant:

```
"What did we find about authentication last time?"
→ Claude calls rlm_search_rag, returns previous analysis

"Show me the security issues we identified"
→ Retrieves past security audit from .rlm/
```

## How It Works

### 1. Automatic Triggering

When you ask a question, Claude:
1. Determines if codebase analysis is needed
2. Calls `rlm_analyze` tool automatically
3. RLM analyzes using trampoline pattern
4. Results returned to Claude
5. Claude synthesizes answer with context

### 2. RAG Storage

Each analysis is stored as:

```json
{
  "query": "Find security vulnerabilities",
  "focus": "security",
  "timestamp": "20260113_140532",
  "result": {
    "type": "RESULT",
    "content": "Security analysis findings...",
    "metadata": {...}
  },
  "stats": {
    "total_cost_usd": 0.38,
    "total_tokens": 125000
  },
  "path": "./src"
}
```

### 3. Intelligent Search

When Claude needs past info:
- Searches `.rlm/index.json` for relevant queries
- Scores results by keyword matching
- Returns most relevant past analyses
- No re-analysis needed!

## Tool Reference

### `rlm_analyze`

**Automatically called when Claude needs to:**
- Understand codebase structure
- Find patterns across files
- Analyze security, performance, architecture
- Review documentation
- Explain complex systems

**Parameters:**
- `path` - Directory or file to analyze (default: current dir)
- `query` - What to find/analyze
- `focus` - Optional: `security`, `architecture`, `performance`, `documentation`, `testing`, `general`

**Example invocation by Claude:**
```json
{
  "tool": "rlm_analyze",
  "arguments": {
    "path": ".",
    "query": "Find all authentication mechanisms and assess security",
    "focus": "security"
  }
}
```

### `rlm_search_rag`

**Automatically called when Claude needs to recall:**
- Previous analysis results
- Past findings without re-analyzing
- Historical insights

**Parameters:**
- `query` - What to search for
- `max_results` - How many results (default: 5)

**Example invocation by Claude:**
```json
{
  "tool": "rlm_search_rag",
  "arguments": {
    "query": "security vulnerabilities",
    "max_results": 3
  }
}
```

### `rlm_status`

**Manually invoked or when Claude checks progress:**
- Shows current analysis status
- Displays costs and metrics
- Checks recursion depth

## Example Workflow

### First Analysis

**You:** "Analyze this codebase for security issues"

**Claude thinks:** *This requires analyzing multiple files, I'll use RLM*

```
🔧 Using tool: rlm_analyze
   path: "."
   query: "Find security vulnerabilities"
   focus: "security"

🚀 RLM analyzing...
   Explorer: Identifying files...
   Worker: Analyzing auth.py...
   Worker: Analyzing database.py...
   Worker: Analyzing api.py...

✅ Analysis complete!
   Cost: $2.34
   Issues found: 5
   Stored in: .rlm/analysis_20260113_140532.json
```

**Claude:** "I found 5 security issues:
1. SQL injection risk in database.py:45
2. Missing auth check in api.py:120
..."

### Later Retrieval

**You:** "What security issues did you find earlier?"

**Claude thinks:** *This is asking about past analysis, I'll search RAG*

```
🔧 Using tool: rlm_search_rag
   query: "security issues"
   max_results: 3

✅ Found 1 relevant analysis from 2 hours ago
   Query: "Find security vulnerabilities"
   Cost: $0.00 (cached)
```

**Claude:** "Earlier I found 5 security issues:
1. SQL injection risk in database.py:45
..."

## Configuration

### Per-Repo Settings

Create `.rlm/config.json` in your project:

```json
{
  "max_recursion_depth": 10,
  "cache_enabled": true,
  "cache_ttl_hours": 168,
  "auto_focus_detection": true,
  "excluded_paths": [
    "node_modules/",
    "venv/",
    ".git/",
    "*.log"
  ]
}
```

### Global Settings

Edit `~/.claude-rlm/config.json`:

```json
{
  "max_recursion_depth": 10,
  "cache_enabled": true,
  "cache_ttl_hours": 24,
  "cost_tracking": true,
  "max_cost_per_analysis": 10.0
}
```

## Cost Management

### Viewing Costs

Check `.rlm/index.json` to see analysis costs:

```json
[
  {
    "timestamp": "20260113_140532",
    "query": "Security audit",
    "focus": "security",
    "file": "analysis_20260113_140532.json",
    "path": ".",
    "cost_usd": 2.34,
    "tokens": 780000
  }
]
```

### Cost Optimization

**Caching works automatically:**
- Identical queries: ~90% cost reduction
- Similar queries: ~70% cost reduction
- RAG retrieval: $0.00 (no re-analysis)

**Tips:**
1. **Be specific** - "Find SQL injection" vs "Analyze security"
2. **Use focus** - Narrows analysis scope
3. **Search RAG first** - Check if question answered before
4. **Progressive queries** - Start broad, then narrow

## Troubleshooting

### RLM not triggering

**Check:**
1. MCP server connected (look for indicator in Claude)
2. Path in config is absolute and correct
3. Python 3.8+ installed: `python3 --version`

**Test manually:**
```bash
cd claude-rlm
python3 mcp_server/rlm_server.py
# Should show: "RLM MCP Server starting on stdio..."
```

### "No previous analyses found"

**Cause:** `.rlm/` directory doesn't exist yet

**Solution:** Run an analysis first:
```
"Analyze this codebase for [something]"
```

### High costs

**Check:**
1. Query too broad - be more specific
2. Many large files - use `excluded_paths`
3. No caching - ensure `cache_enabled: true`

**View stats:**
```bash
cat .rlm/index.json | jq '.[].cost_usd'
```

### Permission errors

**Unix/Mac:**
```bash
chmod +x mcp_server/rlm_server.py
```

**Windows:** Run as administrator or check Python permissions

## Advanced Usage

### Custom Focus Areas

Add to `.rlm/config.json`:

```json
{
  "custom_focus_areas": {
    "api_security": {
      "patterns": ["auth", "token", "api", "endpoint"],
      "file_types": [".py", ".js", ".ts"]
    },
    "database": {
      "patterns": ["sql", "query", "database", "orm"],
      "file_types": [".py", ".sql"]
    }
  }
}
```

### Webhook Integration

Trigger analysis on git push:

```bash
# .git/hooks/post-commit
#!/bin/bash
python3 ~/claude-rlm/mcp_server/cli.py analyze . "Review recent changes"
```

### CI/CD Integration

```yaml
# .github/workflows/rlm-analysis.yml
name: RLM Security Audit
on: [pull_request]
jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run RLM
        run: |
          python3 ~/claude-rlm/src/orchestrator.py . "Security audit of PR changes"
```

## Privacy & Security

### What's Stored

- **Local only**: All data in `.rlm/` stays in your repo
- **No cloud**: RLM runs locally, no external API calls
- **Git-ignored**: `.rlm/` automatically excluded from commits

### Sensitive Data

Add to `.gitignore`:
```
.rlm/
```

Or encrypt results:
```bash
# Encrypt before committing
gpg --encrypt .rlm/analysis_*.json
```

## Summary

**Install:** Add MCP config → Restart Claude
**Use:** Just ask questions naturally
**RAG:** Results stored in `.rlm/` automatically
**Retrieve:** Claude searches RAG when relevant
**Cost:** Caching makes repeat queries nearly free

**No commands needed!** Claude handles everything automatically.

---

For more details, see:
- Main README: [README.md](README.md)
- Orchestrator docs: [.claude/CLAUDE.md](.claude/CLAUDE.md)
- RLM patterns: [skills/rlm-patterns/SKILL.md](skills/rlm-patterns/SKILL.md)
