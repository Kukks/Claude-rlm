---
name: rlm:status
description: Check status of current or recent RLM analysis
---

# RLM Status Command

## Usage

```bash
/rlm:status
```

## What It Does

Displays the current state of RLM analysis, including:
- Analysis status (idle, active, completed)
- Current recursion depth
- Stack size (number of pending parent tasks)
- Results collected so far
- Performance statistics
- Cost tracking

## Output Format

### When No Analysis is Running

```
RLM Status: Idle
No active analysis
```

### When Analysis is Active

```
RLM Status: Active
================

Current Task:
  Agent: Worker
  Depth: 3/10
  Task: Analyze authentication module for security issues

Stack Info:
  Stack depth: 2
  Pending parent tasks: 2

Progress:
  Subagent calls: 15
  Results collected: 12
  Cache hits: 3

Performance:
  Total tokens: 125,000
  Total cost: $0.38
  Max depth reached: 3

Started: 2026-01-13T14:30:00Z
```

### When Analysis is Completed

```
RLM Status: Completed
===================

Last Analysis Summary:
  Total subagent calls: 25
  Max depth reached: 4
  Cache hits: 8 (32% hit rate)
  Total tokens: 250,000
  Total cost: $0.75
  Duration: 12 minutes

Analysis completed at: 2026-01-13T14:42:00Z
```

## Use Cases

1. **Monitor Progress**: Check how far along a long-running analysis is
2. **Debug Issues**: See if analysis is stuck at a particular depth
3. **Cost Tracking**: Monitor spending in real-time
4. **Resume Decision**: Decide whether to resume or restart

## Implementation

The command:
1. Loads `.rlm_state.json` if it exists
2. Parses the state information
3. Formats and displays human-readable status
4. Returns exit code 0 for active analysis, 1 for idle

## Example Usage

```bash
# Start an analysis
/rlm:analyze ./large_codebase "Security audit"

# Check status while it runs
/rlm:status
# Shows: Active, depth 2, 10 subagent calls...

# Resume later
/rlm:analyze
# Continues from where it left off

# Check final status
/rlm:status
# Shows: Completed, 25 total calls, $0.75 cost
```

## Tips

- Use `/rlm:status` frequently during long analyses to monitor progress
- If stack depth is unusually high, consider simplifying your query
- High cache hit rates (>50%) indicate efficient reuse of results
- Monitor cost in real-time to stay within budget
