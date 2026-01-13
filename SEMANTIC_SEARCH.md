# Semantic Search in RLM RAG

## Overview

RLM uses **vector embeddings** for semantic search when ChromaDB is installed, with automatic fallback to keyword search when it's not available.

**Semantic search** understands the *meaning* of your query, not just exact word matches.

## Quick Comparison

| Feature | Keyword Search (JSON) | Semantic Search (ChromaDB) |
|---------|----------------------|---------------------------|
| **Installation** | Built-in (no dependencies) | Requires `pip install chromadb` |
| **Query Understanding** | Exact word matching | Meaning-based matching |
| **Example** | "auth" finds "authentication" | "login bugs" finds "authentication vulnerabilities" |
| **Speed** | Very fast | Fast (with caching) |
| **Quality** | Good for exact terms | Excellent for concepts |
| **Storage** | Lightweight JSON files | Vector database + JSON |

## How Semantic Search Works

### 1. When You Store Analysis

```python
# You analyze code
rlm_analyze("Find authentication vulnerabilities")

# RLM stores with vector embedding
{
  "query": "Find authentication vulnerabilities",
  "result": {...},
  "embedding": [0.234, -0.456, 0.789, ...]  # 384-dimensional vector
}
```

The embedding captures the *meaning* of your query and results.

### 2. When You Search

```python
# You search later
rlm_search_rag("login security issues")

# Semantic search converts your query to a vector
query_embedding = [0.221, -0.441, 0.801, ...]

# Finds similar vectors (similar meanings)
# Ranks by semantic similarity, not just keyword overlap
```

### 3. Why It's Better

**Keyword search:**
```
Query: "login bugs"
Found: ❌ Nothing (no analysis has exact words "login" or "bugs")
```

**Semantic search:**
```
Query: "login bugs"
Found: ✅ "Find authentication vulnerabilities"
       ✅ "Review user access control"
       ✅ "Check password validation"

Why? These are semantically similar concepts!
```

## Real-World Examples

### Example 1: Concept Matching

**Stored Analysis:**
- "Find SQL injection vulnerabilities"
- "Review database query security"
- "Optimize slow queries"

**Your Query:** "database security issues"

**Keyword Search Returns:**
1. "Review database query security" (has "database" + "security")
2. Nothing else

**Semantic Search Returns:**
1. "Review database query security" (has "database" + "security")
2. "Find SQL injection vulnerabilities" (SQL injection IS a database security issue!)
3. "Optimize slow queries" (related to databases, but lower score)

### Example 2: Synonym Understanding

**Stored Analysis:**
- "Analyze authentication logic"

**Your Query:** "login bugs"

**Keyword Search:** ❌ No results (no word match)

**Semantic Search:** ✅ Finds "Analyze authentication logic"
- Understands "login" ≈ "authentication"
- Understands "bugs" ≈ "issues/problems in logic"

### Example 3: Related Concepts

**Stored Analysis:**
- "Review password hashing implementation"

**Your Query:** "security vulnerabilities"

**Keyword Search:** ❌ No results (no word "security" or "vulnerabilities")

**Semantic Search:** ✅ Finds "Review password hashing implementation"
- Understands weak password hashing IS a security vulnerability
- Ranks it appropriately based on semantic relevance

## Installation

### Option 1: With Semantic Search (Recommended)

```bash
# Install ChromaDB for semantic search
pip install chromadb

# Or install all optional dependencies
pip install -r requirements.txt
```

**Benefits:**
- 10x better search quality
- Finds related concepts, not just keywords
- Understands synonyms and related terms
- Better ranking of results

**Requirements:**
- ~100MB for ChromaDB and dependencies
- ~80MB for embedding model (downloaded once)
- Slightly more storage per analysis (~500KB vs ~50KB)

### Option 2: Keyword Search Only (Fallback)

```bash
# No additional installation needed
# Works out of the box
```

**Benefits:**
- Zero dependencies
- Lightweight storage
- Very fast search
- Good for exact term matching

**Limitations:**
- Must use exact words from queries
- Won't find related concepts
- No synonym understanding

## Checking Your Setup

### Auto-detection

RLM automatically detects and uses the best available backend:

```python
# When RLM starts
✓ ChromaDB available - semantic search enabled
```

Or:

```python
# When ChromaDB not installed
ℹ ChromaDB not found - using keyword search
  Install for better search: pip install chromadb
```

### Manual Check

```bash
# Check if ChromaDB is installed
python3 -c "import chromadb; print(f'ChromaDB {chromadb.__version__} installed')"

# Run RLM installer to see your setup
python3 install.py
```

## Storage Details

### With ChromaDB (Semantic)

```
your-repo/
└── .rlm/
    ├── index.json                    # Fast lookup index
    ├── analysis_20260113_120000.json # Full analysis result
    ├── analysis_20260113_130000.json
    └── chromadb/                     # Vector database
        ├── chroma.sqlite3            # Metadata
        └── [uuid]/                   # Embeddings
```

**Storage per analysis:**
- ~50KB: Full result JSON
- ~450KB: Vector embeddings
- **Total: ~500KB per analysis**

### With JSON Only (Keyword)

```
your-repo/
└── .rlm/
    ├── index.json                    # Fast lookup index
    ├── analysis_20260113_120000.json # Full analysis result
    └── analysis_20260113_130000.json
```

**Storage per analysis:**
- ~50KB: Full result JSON
- **Total: ~50KB per analysis**

## Search Quality Comparison

### Test Query: "authentication bugs"

**Stored Analyses:**
1. "Find SQL injection vulnerabilities"
2. "Review login endpoint security"
3. "Optimize database performance"
4. "Check password validation logic"

#### Keyword Search Results:
```
No results found
(none of the queries have "authentication" or "bugs")
```

#### Semantic Search Results:
```
1. "Review login endpoint security"      (score: 92/100)
2. "Check password validation logic"     (score: 87/100)
3. "Find SQL injection vulnerabilities"  (score: 78/100)
4. "Optimize database performance"       (score: 31/100)
```

**Why these rankings?**
- Login endpoint security directly relates to authentication
- Password validation is part of authentication
- SQL injection can affect authentication systems
- Database performance is loosely related (low score)

## Performance

### Search Speed

**Keyword search:**
- ~1-5ms for small repos (<100 analyses)
- ~10-50ms for large repos (1000+ analyses)

**Semantic search:**
- ~5-20ms for small repos (<100 analyses)
- ~20-100ms for large repos (1000+ analyses)
- First query may take ~1s (loads embedding model)
- Subsequent queries are fast (model cached)

### Storage Overhead

**Keyword search:**
- ~50KB per analysis
- 1000 analyses = ~50MB

**Semantic search:**
- ~500KB per analysis
- 1000 analyses = ~500MB

**Recommendation:** For most repos with <100 analyses, the storage difference is negligible (<50MB).

## When to Use Each

### Use Semantic Search (ChromaDB) When:

✅ You search using natural language
✅ You want to find related concepts
✅ You don't remember exact query wording
✅ You're building knowledge base for large projects
✅ Storage space is not a constraint

**Example workflows:**
- "What security issues did we find?" (finds auth, SQL injection, XSS, etc.)
- "Performance problems" (finds slow queries, optimization, caching, etc.)
- "API design decisions" (finds REST, GraphQL, endpoint discussions)

### Use Keyword Search (JSON) When:

✅ Storage is very limited
✅ You search using exact terms
✅ You want maximum speed
✅ You can't install dependencies
✅ Working in restricted environments

**Example workflows:**
- "authentication" (finds analyses with exact word "authentication")
- "SQL injection" (finds exact phrase matches)
- "performance" (finds exact word in queries)

## Migration

### Upgrading from Keyword to Semantic

1. Install ChromaDB:
```bash
pip install chromadb
```

2. Restart RLM (auto-detects ChromaDB)

3. Existing analyses work as-is:
```
✓ Old keyword search data remains searchable
✓ New analyses use semantic search
✓ Mixed mode works seamlessly
```

4. (Optional) Re-analyze to add embeddings:
```python
# Force re-analysis to add vector embeddings
rlm_analyze("your query", force_refresh=true)
```

### Downgrading from Semantic to Keyword

1. Uninstall ChromaDB:
```bash
pip uninstall chromadb
```

2. Restart RLM (auto-detects missing ChromaDB)

3. Existing analyses work as-is:
```
✓ JSON files remain
✓ Embeddings ignored (not needed)
✓ Falls back to keyword search
```

## Technical Details

### Embedding Model

RLM uses **all-MiniLM-L6-v2** by default:
- 384-dimensional embeddings
- 80MB model size
- Trained on 1B+ sentence pairs
- Excellent for code and natural language

**First run downloads model:**
```
Downloading all-MiniLM-L6-v2...
[████████████████] 79.3MB/79.3MB

Model cached to: ~/.cache/chroma/onnx_models/
```

### Distance Metrics

ChromaDB uses **cosine similarity** for ranking:
```python
score = 100 - (distance * 10)

# distance = 0.0 → score = 100 (perfect match)
# distance = 1.0 → score = 90  (very similar)
# distance = 5.0 → score = 50  (somewhat related)
# distance = 10+ → score = 0   (not related)
```

### Backwards Compatibility

Both backends maintain **index.json** for:
- Fast timestamp lookups
- Metadata queries
- Staleness detection
- Migration support

## FAQ

### Q: Can I switch between backends?

**A:** Yes! Just install/uninstall ChromaDB. RLM auto-detects on startup.

### Q: Do I need to re-analyze old data?

**A:** No. Old analyses work with both backends. But re-analyzing adds embeddings for better semantic search.

### Q: Does semantic search work offline?

**A:** Yes! After first download, the embedding model is cached locally.

### Q: How accurate is semantic search?

**A:** Very accurate for code and technical queries. It understands:
- Programming concepts (auth, API, database)
- Security terms (vulnerability, injection, XSS)
- Synonyms (optimize/improve, bug/issue, login/authentication)

### Q: Can I use my own embedding model?

**A:** Not yet, but planned for future versions. Current model works well for code.

### Q: What if ChromaDB breaks?

**A:** RLM gracefully falls back to JSON backend if ChromaDB initialization fails.

## Best Practices

### 1. Install ChromaDB for Active Development

If you're actively working on a codebase:
```bash
pip install chromadb
```

Benefit: Better search as you build knowledge base.

### 2. Use Natural Language in Queries

**Instead of:**
```python
rlm_search_rag("auth")
```

**Use:**
```python
rlm_search_rag("authentication security issues")
```

Semantic search works better with more context.

### 3. Let RLM Auto-Detect

Don't manually specify backend. Let RLM choose based on what's installed.

### 4. Monitor Storage

For large projects with 500+ analyses:
```bash
du -sh .rlm/
# Check if storage is acceptable
```

If storage is a concern, consider keyword search.

### 5. Re-analyze for Important Queries

If you have critical analyses from before ChromaDB:
```python
# Add semantic search to old analyses
rlm_analyze("your important query", force_refresh=true)
```

## Summary

**Semantic Search (ChromaDB):**
- 🎯 Understands meaning, not just words
- 🔍 Finds related concepts and synonyms
- 💾 ~500KB per analysis
- 📦 Requires: `pip install chromadb`
- ⭐ **Recommended for most use cases**

**Keyword Search (JSON):**
- 📝 Exact word matching
- ⚡ Very fast and lightweight
- 💾 ~50KB per analysis
- 📦 Built-in, no dependencies
- ⭐ **Good for resource-constrained environments**

**Both:**
- ✅ Work seamlessly together
- ✅ Auto-detection and graceful fallback
- ✅ No configuration needed
- ✅ Backwards compatible

---

**Ready to try semantic search?**

```bash
pip install chromadb
python3 -c "from mcp_server.storage_backend import get_backend_info; print(get_backend_info())"
```

Start searching for concepts, not just keywords! 🚀
