# RAG Staleness Detection & Claude-mem Integration

## Overview

**Version 2.0** adds:
1. **Automatic staleness detection** - Tracks file changes with SHA256 hashing
2. **Optional Claude-mem integration** - Cross-project learning (if available)
3. **Smart cache invalidation** - Re-analyze only when files change

## Staleness Detection

### How It Works

Every analysis now stores file hashes:

```json
{
  "query": "Find security vulnerabilities",
  "result": {...},
  "file_hashes": {
    "src/auth.py": "a3b2c1...",
    "src/database.py": "d4e5f6...",
    "src/api.py": "g7h8i9..."
  },
  "version": "2.0"
}
```

When you query RAG data, RLM:
1. Computes current file hashes
2. Compares with stored hashes
3. Detects changed, new, or deleted files
4. Warns if data is stale

### New Tool: `rlm_check_freshness`

**Check if cached data is current:**

```
You: "Is the cached analysis still valid?"

Claude: [Calls rlm_check_freshness]

Response:
{
  "fresh": false,
  "staleness_info": {
    "stale": true,
    "changed_files": ["src/auth.py", "src/api.py"],
    "new_files": ["src/oauth.py"],
    "deleted_files": [],
    "total_changes": 3,
    "last_analysis": "20260113_140532",
    "recommendation": "Re-analyze to update"
  }
}
```

### Automatic Detection

**When analyzing:**
```
You: "Find security issues"

RLM: [Checks freshness]
     [Finds 3 files changed]
     [Automatically re-analyzes]

     "Re-analyzing due to file changes:
      - src/auth.py (modified)
      - src/api.py (modified)
      - src/oauth.py (new)"
```

**When searching:**
```
You: "What security issues did we find?"

RLM: [Searches .rlm/]
     [Detects staleness]

     "⚠️ Warning: Data may be outdated.
      3 files changed since last analysis.

      [Returns cached results anyway]

      Recommendation: Run new analysis"
```

### Configuration

Control staleness behavior in `.rlm/config.json`:

```json
{
  "staleness": {
    "enabled": true,
    "auto_reanalyze": false,
    "warn_threshold": 1,
    "excluded_patterns": [
      "*.log",
      "*.tmp",
      "__pycache__"
    ]
  }
}
```

**Options:**
- `enabled`: Track file changes (default: true)
- `auto_reanalyze`: Automatically re-analyze on staleness (default: false)
- `warn_threshold`: Warn if N+ files changed (default: 1)
- `excluded_patterns`: Ignore these files for staleness

## Claude-mem Integration

### What Is Claude-mem?

[Claude-mem](https://github.com/thedotmack/claude-mem) is a memory plugin that stores observations across sessions. RLM can optionally integrate with it for:

1. **Cross-project pattern learning**
2. **Semantic search with embeddings**
3. **Success pattern recommendations**
4. **Cost prediction based on history**

### Detection

RLM automatically detects if claude-mem is available:

```
RLM MCP Server starting on stdio...
Change detection: enabled
Claude-mem integration: enabled  ← Detected!
```

Or:

```
RLM MCP Server starting on stdio...
Change detection: enabled
Claude-mem integration: disabled  ← Not found
```

**No configuration needed!** Works if available, gracefully degrades if not.

### How It Works

#### Before Analysis

**Without claude-mem:**
```
You: "Security audit this Python repo"
RLM: [Starts analysis]
```

**With claude-mem:**
```
You: "Security audit this Python repo"

Claude-mem: "For Python security audits:
             - regex_filter pattern worked well (3 past successes)
             - Average cost: $2.34
             - Found SQL injection in 2/3 projects"

RLM: [Uses recommended pattern]
     [Runs analysis]
```

#### After Analysis

**Store pattern to claude-mem:**
```json
{
  "observation": "RLM security audit success",
  "pattern": "regex_filter",
  "language": "Python",
  "files_analyzed": 45,
  "cost_usd": 2.10,
  "issues_found": 5,
  "success": true
}
```

**Next time on similar project:**
```
Claude-mem suggests: "Try regex_filter - worked on 4 similar repos"
```

### Enhanced Search

#### Repo-local only:
```
You: "What auth issues did we find?"

RLM: [Searches .rlm/ - keyword matching]
     "Found 1 result: Security audit from 2 days ago"
```

#### With claude-mem:
```
You: "What auth issues did we find?"

RLM: [Searches .rlm/ + claude-mem]

     Local results:
     - "Security audit from 2 days ago"

     Claude-mem insights:
     - "Similar auth issues found in 3 other projects"
     - "Common pattern: Missing token expiration"
     - "Recommended fix: Add JWT expiry check"
```

### Installation

**RLM works standalone** - claude-mem is optional!

**To enable claude-mem integration:**

1. Install claude-mem:
```bash
npm install -g claude-mem
```

2. Add to Claude config:
```json
{
  "mcpServers": {
    "claude-mem": {
      "command": "claude-mem",
      "args": ["server"]
    },
    "rlm": {
      "command": "python3",
      "args": ["/path/to/claude-rlm/mcp_server/rlm_server.py"]
    }
  }
}
```

3. Restart Claude Desktop

RLM will automatically detect and integrate!

## Workflow Examples

### Example 1: Fresh Analysis

```
Day 1:
You: "Find security vulnerabilities"
RLM: [Analyzes, stores with file hashes]
     "Found 5 issues. Cached in .rlm/"

Day 2:
You: "What security issues?"
RLM: [Checks freshness: NO changes]
     [Returns cached results instantly]
     "$0 cost, <1 second"
```

### Example 2: Stale Detection

```
Day 1:
You: "Find security vulnerabilities"
RLM: [Analyzes src/]
     "Found 5 issues in auth.py, api.py"

Day 2:
[Developer fixes auth.py, adds oauth.py]

You: "What security issues?"
RLM: [Detects staleness]
     "⚠️ Warning: 2 files changed since last analysis
      - auth.py (modified)
      - oauth.py (new)

      Cached results may be outdated.

      Previous findings:
      - Issue 1 in auth.py (POSSIBLY FIXED)
      - Issue 2 in api.py (likely still valid)
      ...

      Recommendation: Re-analyze"

You: "Re-analyze"
RLM: [force_refresh=true]
     "Updated analysis:
      - auth.py issues FIXED ✓
      - oauth.py: 1 new issue found
      - api.py: still has 1 issue"
```

### Example 3: With Claude-mem

```
Project A (Python Flask):
You: "Security audit"
RLM: [Analyzes, finds SQL injection]
     [Stores pattern to claude-mem]

Project B (Python Django):
You: "Security audit"
Claude-mem: "Tip: Similar Flask project had SQL injection.
             Check query building in views.py"
RLM: [Focuses on query building]
     [Finds issue faster!]
```

## Benefits

### Staleness Detection

**Problem Solved:**
```
# Before: Stale cache
You: "Any security issues?"
RLM: "No issues found"
     [But auth.py was rewritten yesterday! 🐛]
```

```
# After: Staleness detection
You: "Any security issues?"
RLM: "⚠️ auth.py changed - cached data outdated
     Please re-analyze or use with caution"
```

**Accuracy:** Never rely on outdated analysis
**Efficiency:** Only re-analyze when needed
**Transparency:** Know when data is stale

### Claude-mem Integration

**Problem Solved:**
```
# Without claude-mem
Project 1: "Security audit" → $5, finds pattern
Project 2: "Security audit" → $5, finds SAME pattern
Project 3: "Security audit" → $5, finds SAME pattern
Total: $15, learned nothing
```

```
# With claude-mem
Project 1: "Security audit" → $5, learns pattern
Project 2: "Security audit" → $2, uses learned pattern
Project 3: "Security audit" → $1, pattern is now optimized
Total: $8, saved 47%
```

**Learning:** Patterns improve over time
**Efficiency:** Faster analysis on similar projects
**Intelligence:** Cross-project insights

## API Reference

### rlm_check_freshness

**Check if cached data is current**

```json
{
  "tool": "rlm_check_freshness",
  "arguments": {
    "path": "."  // Optional, default current dir
  }
}
```

**Returns:**
```json
{
  "success": true,
  "fresh": false,
  "staleness_info": {
    "stale": true,
    "changed_files": ["file1.py", "file2.py"],
    "new_files": ["file3.py"],
    "deleted_files": ["old.py"],
    "total_changes": 4,
    "last_analysis": "20260113_140532",
    "recommendation": "Re-analyze to update"
  }
}
```

### rlm_analyze (enhanced)

**New parameter: `force_refresh`**

```json
{
  "tool": "rlm_analyze",
  "arguments": {
    "path": ".",
    "query": "Find security vulnerabilities",
    "force_refresh": true  // Re-analyze even if fresh
  }
}
```

**Returns (enhanced):**
```json
{
  "success": true,
  "result": {...},
  "files_tracked": 45,
  "change_detection_enabled": true,
  "staleness_info": {
    "stale": true,
    "changed_files": ["auth.py"]
  },
  "reason_for_analysis": "Files changed since last analysis",
  "claude_mem_suggestions": {
    "pattern_suggestions": ["regex_filter"],
    "estimated_cost": 2.34
  }
}
```

### rlm_search_rag (enhanced)

**New parameter: `include_stale`**

```json
{
  "tool": "rlm_search_rag",
  "arguments": {
    "query": "security issues",
    "include_stale": true  // Default: true (with warning)
  }
}
```

**Returns (enhanced):**
```json
{
  "success": true,
  "results": [...],
  "staleness_info": {
    "stale": true,
    "total_changes": 3
  },
  "warning": "Data may be outdated. 3 files changed since last analysis.",
  "claude_mem_results": {
    "similar_analyses": [...],
    "pattern_suggestions": [...]
  }
}
```

## Migration from v1.0

### Automatic Migration

Old analyses (without file hashes) still work:

```json
// Old format (v1.0)
{
  "query": "Find issues",
  "result": {...}
  // No file_hashes
}
```

**Behavior:**
- Marked as "potentially stale"
- Recommendation: "Re-analyze to enable change detection"
- Still searchable and usable

### Force Re-analysis

To enable change detection on old analyses:

```
You: "Re-analyze with change tracking"
RLM: [force_refresh=true]
     [Stores with file_hashes]
     "Now tracking 45 files for changes"
```

## Troubleshooting

### "no_hash_tracking" warning

**Cause:** Analysis from v1.0 (before staleness detection)

**Solution:**
```
force_refresh=true to re-analyze with hashing
```

### Claude-mem not detected

**Check:**
```bash
which claude-mem
# Should return path if installed
```

**Install:**
```bash
npm install -g claude-mem
```

**Not required!** RLM works fine without it.

### False positives

**Symptom:** Files marked as changed but weren't

**Cause:** File modification time changed (e.g., git checkout)

**Solution:** Hashes are content-based, not mtime-based. If hash matches, file is unchanged.

## Summary

**Staleness Detection:**
- ✅ Tracks file changes with SHA256 hashes
- ✅ Warns when cached data is outdated
- ✅ Smart re-analysis only when needed
- ✅ Transparent about data freshness

**Claude-mem Integration:**
- ✅ Optional - works if available
- ✅ Cross-project pattern learning
- ✅ Better search with embeddings
- ✅ Cost optimization over time

**Both features:**
- ✅ Zero configuration required
- ✅ Automatic detection and activation
- ✅ Graceful degradation if unavailable
- ✅ Backward compatible with v1.0

Ready to use with `claude/build-rlm-orchestrator-dw3se`!
