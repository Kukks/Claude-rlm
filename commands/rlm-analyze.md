---
name: rlm:analyze
description: Start or resume recursive document analysis
---

# RLM Analyze Command

## Usage

```bash
# Start new analysis
/rlm:analyze <document_path> "<query>"

# Resume existing analysis
/rlm:analyze

# Example
/rlm:analyze ./src "Find all security vulnerabilities in the authentication code"
```

## What It Does

1. Checks for existing `.rlm_state.json` (resume capability)
2. Initializes or resumes RLM orchestrator
3. Spawns Explorer subagent to analyze structure
4. Manages trampoline loop for recursive analysis
5. Aggregates results and generates final report
6. Tracks costs and performance metrics

## Parameters

- `document_path`: Path to file or directory to analyze
  - Supports: `.rs`, `.py`, `.js`, `.ts`, `.md`, `.pdf`, `.txt`, and directories
- `query`: Analysis objective (required for new analysis)

## Output

The command produces:
- Real-time progress updates
- Subagent spawn notifications
- Depth and stack size tracking
- Final analysis report
- Cost and performance statistics

## State Management

Analysis state is saved to `.rlm_state.json` after each subagent call, enabling:
- Pause/resume capability
- Crash recovery
- Session persistence

## Examples

### Analyze Codebase
```bash
/rlm:analyze ./src "Find all error handling patterns and assess their robustness"
```

### Resume Interrupted Analysis
```bash
# Previous analysis was interrupted at depth 3
/rlm:analyze
# Continues from where it left off
```

### Security Audit
```bash
/rlm:analyze ./contracts "Security audit: Find vulnerabilities in smart contracts"
```

### Documentation Review
```bash
/rlm:analyze ./docs "What topics are missing from the documentation?"
```

## Cost Estimates

Based on document size:
- Small (<100K tokens): $0.50 - $2.00
- Medium (100K-1M tokens): $2.00 - $10.00
- Large (1M-10M tokens): $10.00 - $50.00

Actual costs depend on:
- Query complexity
- Recursion depth
- Cache hit rate
- Document structure

## Tips

1. **Start Specific**: Narrow queries cost less than broad exploration
2. **Use Filters**: Add keywords to focus analysis
3. **Enable Caching**: Reanalyzing similar documents is nearly free
4. **Check Status**: Use `/rlm:status` to monitor progress
5. **Resume Often**: Large analyses benefit from incremental progress

## Troubleshooting

**"Max recursion depth exceeded"**
- Increase `max_recursion_depth` in config
- Or: Narrow your query to reduce complexity

**"State file corrupted"**
- Delete `.rlm_state.json` and restart
- Previous progress will be lost

**"High cost warning"**
- Query is too broad
- Enable filtering in Explorer agent
- Use more specific keywords

## Implementation

When invoked, this command:
1. Imports the RLM orchestrator from `src/orchestrator.py`
2. Checks for existing state file
3. Either resumes or starts new analysis
4. Runs the trampoline loop until completion
5. Displays final results and statistics
