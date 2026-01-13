---
name: rlm-patterns
description: Library of Recursive Language Model analysis patterns. Reference when designing analysis strategies for large documents.
---

# RLM Patterns Library

This skill contains proven patterns for recursive document analysis, derived from the RLM paper and empirical testing.

## Core Patterns

### 1. Peek Before Processing

**When:** Starting any document analysis
**Purpose:** Understand structure before committing resources

```python
# Always start with a peek
preview = document[:2000]  # First 2000 chars
structure = analyze_structure(preview)

# Make informed decision
if structure.type == "code":
    strategy = "file_by_file"
elif structure.type == "academic_paper":
    strategy = "section_by_section"
```

**Cost:** ~50 tokens
**Benefit:** Prevents wasted analysis on inappropriate strategies

### 2. Partition and Map

**When:** Document has clear boundaries (chapters, files, modules)
**Purpose:** Parallel analysis of independent sections

```python
# Partition
sections = split_by_boundaries(document)

# Map (spawn Workers)
analyses = []
for section in sections:
    result = spawn_worker(f"Analyze: {section.title}", section.content)
    analyses.append(result)

# Reduce
final_analysis = synthesize(analyses)
```

**Best for:** Codebases, multi-chapter documents, collections
**Parallelization:** High (sections are independent)

### 3. Regex-Based Filtering

**When:** Searching for specific patterns or keywords
**Purpose:** Reduce token consumption by filtering irrelevant content

```python
import re

# Define patterns
patterns = {
    "security": r"(auth|token|password|crypto|encrypt)",
    "errors": r"(error|exception|panic|unwrap)",
    "async": r"(async|await|future|tokio)"
}

# Filter content
relevant_sections = []
for name, pattern in patterns.items():
    matches = re.finditer(pattern, document, re.IGNORECASE)
    for match in matches:
        # Extract surrounding context (±500 chars)
        start = max(0, match.start() - 500)
        end = min(len(document), match.end() + 500)
        relevant_sections.append({
            "type": name,
            "content": document[start:end],
            "match": match.group()
        })

# Analyze only relevant sections
for section in relevant_sections:
    analyze_section(section)
```

**Token savings:** 70-90% for targeted queries
**Best for:** Finding specific patterns, security audits, API documentation

### 4. Verification Loop

**When:** High-stakes analysis (security, legal, financial)
**Purpose:** Increase confidence through multiple passes

```python
# Pass 1: Quick scan
candidates = quick_scan(content)

# Pass 2: Verify each candidate
verified = []
for candidate in candidates:
    if verify_finding(candidate, content):
        verified.append(candidate)

# Pass 3: Deep analysis of verified
for item in verified:
    detailed_analysis = deep_analyze(item)
    if detailed_analysis.confidence > 0.8:
        report_finding(detailed_analysis)
```

**Cost:** 3x base analysis
**Benefit:** 90%+ reduction in false positives

### 5. Hierarchical Decomposition

**When:** Complex nested structures (class hierarchies, org charts)
**Purpose:** Understand relationships and dependencies

```python
# Build hierarchy
root = parse_structure(document)

# Recursive descent
def analyze_node(node, depth=0):
    analysis = {
        "name": node.name,
        "direct_analysis": analyze_content(node.content),
        "children": []
    }

    for child in node.children:
        if depth < max_depth:
            child_analysis = analyze_node(child, depth + 1)
            analysis["children"].append(child_analysis)

    return analysis

result = analyze_node(root)
```

**Best for:** Inheritance trees, directory structures, nested configurations

### 6. Iterative Refinement

**When:** Initial analysis reveals need for deeper investigation
**Purpose:** Progressive deepening based on findings

```python
# Phase 1: High-level overview
overview = analyze_overview(document)

# Phase 2: Identify interesting sections
hotspots = identify_hotspots(overview)

# Phase 3: Deep dive into hotspots
detailed_findings = []
for hotspot in hotspots:
    if hotspot.complexity > threshold:
        # Spawn Worker for detailed analysis
        detail = spawn_worker("Deep analysis", hotspot.content)
        detailed_findings.append(detail)

# Phase 4: Synthesis
return synthesize_findings(overview, detailed_findings)
```

**Adaptive:** Depth varies based on complexity
**Cost-effective:** Only analyzes what's necessary

### 7. State Machine Pattern

**When:** Analysis spans multiple sessions (very large documents)
**Purpose:** Resumable analysis with checkpointing

```python
# Load or initialize state
state = load_state() or {
    "phase": "structure",
    "sections_completed": 0,
    "total_sections": 0,
    "results": []
}

# Execute current phase
if state["phase"] == "structure":
    structure = analyze_structure()
    state["total_sections"] = len(structure.sections)
    state["phase"] = "analysis"
    save_state(state)

elif state["phase"] == "analysis":
    section = get_next_section(state["sections_completed"])
    result = analyze_section(section)
    state["results"].append(result)
    state["sections_completed"] += 1

    if state["sections_completed"] >= state["total_sections"]:
        state["phase"] = "synthesis"
    save_state(state)

elif state["phase"] == "synthesis":
    final_report = synthesize(state["results"])
    clear_state()
    return final_report
```

**Resumable:** Can pause/resume anytime
**Fault-tolerant:** Survives crashes/disconnections

## Pattern Selection Guide

| Document Type | Recommended Pattern | Reason |
|--------------|-------------------|---------|
| Large codebase (>1M tokens) | Partition + Verification | Files are independent, quality critical |
| Academic paper (50-200K) | Hierarchical Decomposition | Clear section structure |
| Legal document (100-500K) | Regex Filter + Verification | Need specific clauses, high accuracy |
| API documentation (200K) | Regex Filter + Partition | Search-driven, many independent sections |
| Novel/book (500K+) | State Machine | Too large for single session |
| Technical spec (100K) | Iterative Refinement | Complexity varies by section |

## Cost Optimization Strategies

### 1. Early Filtering
Filter irrelevant content before spawning subagents.

**Bad:**
```python
for file in all_files:  # 1000 files
    analyze_file(file)  # $10 total
```

**Good:**
```python
relevant_files = [f for f in all_files if is_relevant(f)]  # 100 files
for file in relevant_files:
    analyze_file(file)  # $1 total
```

### 2. Response Caching
Hash (task + context) and cache results.

```python
cache_key = hash(f"{task}{json.dumps(context)}")
if cached_result := cache.get(cache_key):
    return cached_result

result = perform_analysis()
cache.set(cache_key, result, ttl=24*3600)
return result
```

### 3. Incremental Processing
Stop early if no issues found.

```python
chunks = split_into_chunks(document)
for chunk in chunks:
    issues = scan_for_issues(chunk)
    if not issues:
        continue  # Skip deep analysis

    deep_analysis = analyze_deeply(chunk)
    report_findings(deep_analysis)
```

## Anti-Patterns (Avoid)

### ❌ Exhaustive Processing
```python
# Bad: Processes everything
for line in document.split('\n'):
    analyze_line(line)  # Millions of API calls
```

### ❌ Premature Recursion
```python
# Bad: Recurses on trivial content
if len(section) > 0:
    spawn_worker(section)  # Even for 1 line
```

### ❌ No Early Exit
```python
# Bad: Continues even when done
for section in sections:
    analyze(section)  # Should stop when answer found
```

### ❌ Context Bloat
```python
# Bad: Includes entire document in every context
context = {
    "full_document": document,  # 1M tokens repeated
    "section": current_section
}
```

## Quick Reference

**Starting analysis?** → Peek Before Processing
**Clear boundaries?** → Partition and Map
**Searching for patterns?** → Regex-Based Filtering
**High stakes?** → Verification Loop
**Nested structure?** → Hierarchical Decomposition
**Evolving analysis?** → Iterative Refinement
**Very large doc?** → State Machine Pattern

**Always consider:** Cost, accuracy, time, and resumability.
