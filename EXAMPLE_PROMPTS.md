# Example Prompts - Natural Language RLM Triggers

These prompts will automatically trigger RLM analysis when you have MCP integration enabled.

## Code Understanding

### Architecture & Structure
```
"Explain how this codebase is organized"
"What are the main components of this system?"
"Show me the data flow from API to database"
"How do different modules interact with each other?"
"What design patterns are used in this project?"
```

### Specific Features
```
"How does authentication work in this repo?"
"Explain the payment processing flow"
"How are errors handled throughout the application?"
"What's the caching strategy?"
"How does the state management work?"
```

## Security Analysis

### General Security
```
"Find security vulnerabilities in this codebase"
"Check for common security issues"
"Analyze authentication and authorization mechanisms"
"Are there any security best practices being violated?"
```

### Specific Vulnerabilities
```
"Find all SQL queries and check for injection risks"
"Check for XSS vulnerabilities in the frontend"
"Are API endpoints properly authenticated?"
"Look for hardcoded secrets or credentials"
"Check for CSRF protection"
"Find insecure direct object references"
```

## Code Quality

### Bug Detection
```
"Find potential bugs in error handling"
"Check for race conditions"
"Look for memory leaks"
"Find null pointer dereferences"
"Check for resource leaks (file handles, connections)"
```

### Best Practices
```
"Check code against best practices"
"Find code that needs refactoring"
"Identify overly complex functions"
"Look for duplicated code"
"Check error handling consistency"
```

## Performance Analysis

```
"Find performance bottlenecks"
"Check database query efficiency"
"Look for N+1 query problems"
"Find unnecessary computations"
"Check for proper caching usage"
"Identify slow algorithms"
```

## Documentation Review

```
"What's missing from the documentation?"
"Are all public APIs documented?"
"Check documentation accuracy against code"
"What functions need better comments?"
"Is the README complete?"
```

## Testing Analysis

```
"What test coverage do we have?"
"Find untested critical paths"
"Are edge cases tested?"
"What tests are missing?"
"Check test quality and maintainability"
```

## Dependency Analysis

```
"What external dependencies are used?"
"Check for outdated dependencies"
"Are there unused dependencies?"
"What's the dependency tree for X?"
```

## Migration & Updates

```
"What needs to change to upgrade to Python 3.12?"
"Find all uses of deprecated APIs"
"What would break if we remove feature X?"
"How to migrate from library A to library B?"
```

## Compliance & Standards

```
"Check PCI-DSS compliance"
"Verify GDPR data handling"
"Check against coding standards"
"Find accessibility issues"
"Check license compatibility"
```

## Comparative Analysis

```
"Compare authentication implementation in auth.py vs oauth.py"
"How do the frontend and backend handle errors differently?"
"Compare performance of old vs new implementation"
"What's different between v1 and v2 APIs?"
```

## RAG Retrieval Prompts

These prompts trigger `rlm_search_rag` to retrieve past analyses:

```
"What security issues did we find last time?"
"Show me the previous architecture analysis"
"What were the performance bottlenecks we identified?"
"Remind me what needs refactoring"
"What did the last security audit reveal?"
"Show previous findings about [topic]"
```

## Combined Queries

These demonstrate RLM's ability to handle complex, multi-part analyses:

```
"Find all authentication code, check for vulnerabilities,
and verify it follows OAuth 2.0 best practices"

"Analyze the payment flow from frontend to database,
identify security risks, and check error handling"

"Review all API endpoints, verify authentication,
check for injection vulnerabilities, and validate input handling"

"Find database access code, check for SQL injection,
verify proper connection pooling, and identify N+1 queries"
```

## Progressive Queries

Start broad, then narrow based on results:

```
1. "What are the main security concerns in this codebase?"
   → RLM provides overview

2. "Tell me more about the SQL injection risks you found"
   → Claude retrieves from RAG, no re-analysis

3. "Show me the code for the vulnerable query in user_service.py"
   → Specific file analysis

4. "How should I fix that SQL injection vulnerability?"
   → Recommendation based on context
```

## Focus-Specific Queries

Add focus hints to narrow analysis:

### Security Focus
```
"[Focus: security] Analyze this API module"
"[Focus: security] Review authentication code"
```

### Performance Focus
```
"[Focus: performance] Check database queries"
"[Focus: performance] Analyze this algorithm"
```

### Architecture Focus
```
"[Focus: architecture] Explain the system design"
"[Focus: architecture] How do services communicate?"
```

### Documentation Focus
```
"[Focus: documentation] What's missing from API docs?"
"[Focus: documentation] Check README completeness"
```

## Real-World Examples

### Example 1: New Developer Onboarding
```
"I'm new to this codebase. Explain:
1. Overall architecture
2. How to run locally
3. Where to find key features
4. Coding conventions used"
```

### Example 2: Pre-Deployment Security Check
```
"Before deploying to production:
1. Find security vulnerabilities
2. Check for hardcoded secrets
3. Verify authentication on all endpoints
4. Check error handling exposes no sensitive data"
```

### Example 3: Performance Optimization
```
"Our API is slow. Help me:
1. Find performance bottlenecks
2. Identify slow database queries
3. Check caching implementation
4. Suggest optimization strategies"
```

### Example 4: Code Review Assistance
```
"Review PR #123:
1. Check for security issues
2. Verify error handling
3. Assess code quality
4. Find potential bugs"
```

### Example 5: Technical Debt Assessment
```
"Assess technical debt:
1. Find code that needs refactoring
2. Identify duplicated code
3. Check test coverage
4. Find deprecated API usage"
```

## Tips for Effective Prompts

### ✅ Good Prompts

**Specific:**
```
"Find SQL injection vulnerabilities in the authentication module"
```

**Clear scope:**
```
"Analyze security in src/api/ directory"
```

**Actionable:**
```
"Find and explain all error handling issues"
```

### ❌ Poor Prompts

**Too vague:**
```
"Check everything"  # Too broad, expensive
```

**No context:**
```
"Is it good?"  # What aspect? What criteria?
```

**Multiple unrelated topics:**
```
"Check security, performance, style, and documentation"
# Better: 4 separate analyses
```

## Combining with RAG

### First Time Analysis
```
"Find all authentication mechanisms and assess security"
→ RLM performs full analysis
→ Results stored in .rlm/
```

### Later Retrieval
```
"What authentication mechanisms did you find?"
→ Claude retrieves from .rlm/
→ No re-analysis, instant results, $0 cost
```

### Update Analysis
```
"Re-analyze authentication - we added OAuth support"
→ New analysis
→ Stored alongside previous results
→ Can compare old vs new
```

## Summary

**Key patterns:**
- Be specific about what to find/analyze
- Mention focus area (security, performance, etc.)
- Ask follow-up questions to drill down
- Use RAG retrieval for previously analyzed topics
- Combine multiple aspects in complex queries

**RLM automatically handles:**
- File discovery
- Recursive analysis
- Result aggregation
- RAG storage
- Cost optimization

**Just ask naturally - Claude + RLM handle the rest!**
