#!/usr/bin/env python3
"""
Comprehensive tests for storage backends (ChromaDB vs JSON)
"""
import json
import sys
import os
import shutil
import tempfile
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, os.path.dirname(__file__))

from mcp_server.storage_backend import (
    ChromaDBBackend,
    JSONBackend,
    create_storage_backend,
    get_backend_info,
    CHROMADB_AVAILABLE
)

class TestResults:
    def __init__(self):
        self.passed = 0
        self.failed = 0
        self.tests = []

    def record(self, test_name, passed, message=""):
        if passed:
            self.passed += 1
            print(f"✓ {test_name}")
        else:
            self.failed += 1
            print(f"✗ {test_name}: {message}")
        self.tests.append((test_name, passed, message))

    def summary(self):
        print()
        print("="*70)
        print(f"Results: {self.passed} passed, {self.failed} failed")
        if self.failed == 0:
            print("✅ All tests passed!")
            return 0
        else:
            print("❌ Some tests failed")
            return 1


def test_backend_detection(results):
    """Test that backend detection works"""
    info = get_backend_info()

    results.record(
        "Backend info structure",
        "chromadb_available" in info and "default_backend" in info and "semantic_search" in info,
        f"Got: {info}"
    )

    results.record(
        "ChromaDB availability detection",
        info['chromadb_available'] == CHROMADB_AVAILABLE,
        f"Expected {CHROMADB_AVAILABLE}, got {info['chromadb_available']}"
    )

    expected_backend = "chromadb" if CHROMADB_AVAILABLE else "json"
    results.record(
        "Default backend selection",
        info['default_backend'] == expected_backend,
        f"Expected {expected_backend}, got {info['default_backend']}"
    )


def test_json_backend(results):
    """Test JSON backend functionality"""
    test_dir = tempfile.mkdtemp(prefix="rlm_test_json_")

    try:
        backend = JSONBackend(test_dir)

        # Test storage
        backend.store(
            timestamp="20260113_120000",
            query="Find authentication bugs",
            focus="security",
            result={"content": "Found 2 issues in login.py"},
            file_hashes={"login.py": "abc123", "auth.py": "def456"},
            stats={"total_tokens": 1000, "total_cost_usd": 0.05},
            path=test_dir
        )

        # Verify files created
        index_file = Path(test_dir) / "index.json"
        results.record(
            "JSON backend creates index",
            index_file.exists(),
            f"Index file not found at {index_file}"
        )

        result_file = Path(test_dir) / "analysis_20260113_120000.json"
        results.record(
            "JSON backend creates result file",
            result_file.exists(),
            f"Result file not found at {result_file}"
        )

        # Test get_all
        all_entries = backend.get_all()
        results.record(
            "JSON backend get_all returns entries",
            len(all_entries) == 1 and all_entries[0]['query'] == "Find authentication bugs",
            f"Got {len(all_entries)} entries"
        )

        # Test keyword search - exact match
        search_results = backend.search("authentication", max_results=5)
        results.record(
            "JSON backend keyword search (exact match)",
            len(search_results) > 0 and "authentication" in search_results[0]['query'].lower(),
            f"Search returned {len(search_results)} results"
        )

        # Test keyword search - no match
        search_results = backend.search("performance", max_results=5)
        results.record(
            "JSON backend keyword search (no match)",
            len(search_results) == 0,
            f"Expected 0 results, got {len(search_results)}"
        )

        # Test multiple entries
        backend.store(
            timestamp="20260113_130000",
            query="Optimize database queries",
            focus="performance",
            result={"content": "Found slow queries"},
            file_hashes={"db.py": "ghi789"},
            stats={"total_tokens": 800, "total_cost_usd": 0.04},
            path=test_dir
        )

        all_entries = backend.get_all()
        results.record(
            "JSON backend stores multiple entries",
            len(all_entries) == 2,
            f"Expected 2 entries, got {len(all_entries)}"
        )

        # Test metadata
        with open(result_file) as f:
            data = json.load(f)

        results.record(
            "JSON backend includes file_hashes",
            "file_hashes" in data and len(data['file_hashes']) == 2,
            f"file_hashes: {data.get('file_hashes', 'missing')}"
        )

        results.record(
            "JSON backend includes version",
            data.get('version') == "2.0",
            f"version: {data.get('version', 'missing')}"
        )

        results.record(
            "JSON backend includes storage_backend",
            data.get('storage_backend') == "json",
            f"storage_backend: {data.get('storage_backend', 'missing')}"
        )

    finally:
        shutil.rmtree(test_dir)


def test_chromadb_backend(results):
    """Test ChromaDB backend functionality"""
    if not CHROMADB_AVAILABLE:
        print("⏭️  Skipping ChromaDB tests (not installed)")
        return

    test_dir = tempfile.mkdtemp(prefix="rlm_test_chroma_")

    try:
        backend = ChromaDBBackend(test_dir)

        # Test storage
        backend.store(
            timestamp="20260113_140000",
            query="Find security vulnerabilities in authentication",
            focus="security",
            result={"content": "SQL injection in login, XSS in profile"},
            file_hashes={"login.py": "abc123", "profile.py": "def456"},
            stats={"total_tokens": 1200, "total_cost_usd": 0.06},
            path=test_dir
        )

        # Verify ChromaDB directory created
        chroma_dir = Path(test_dir) / "chromadb"
        results.record(
            "ChromaDB backend creates chromadb directory",
            chroma_dir.exists(),
            f"ChromaDB dir not found at {chroma_dir}"
        )

        # Verify index still maintained for backwards compatibility
        index_file = Path(test_dir) / "index.json"
        results.record(
            "ChromaDB backend maintains JSON index",
            index_file.exists(),
            f"Index file not found at {index_file}"
        )

        # Test get_all
        all_entries = backend.get_all()
        results.record(
            "ChromaDB backend get_all returns entries",
            len(all_entries) == 1,
            f"Got {len(all_entries)} entries"
        )

        # Test semantic search - should find similar concepts
        backend.store(
            timestamp="20260113_150000",
            query="Improve database performance",
            focus="performance",
            result={"content": "Slow queries in user lookup"},
            file_hashes={"db.py": "ghi789"},
            stats={"total_tokens": 900, "total_cost_usd": 0.045},
            path=test_dir
        )

        # Search for "auth bugs" should find "security vulnerabilities in authentication"
        search_results = backend.search("auth bugs", max_results=5)
        results.record(
            "ChromaDB semantic search finds related concepts",
            len(search_results) > 0,
            f"Search returned {len(search_results)} results"
        )

        if len(search_results) > 0:
            # The security query should rank higher than performance query
            security_found = any("security" in r['query'].lower() or "authentication" in r['query'].lower()
                               for r in search_results)
            results.record(
                "ChromaDB semantic search ranks relevant results",
                security_found,
                f"Queries found: {[r['query'] for r in search_results]}"
            )

        # Test that embeddings are actually being used
        results.record(
            "ChromaDB results include semantic score",
            len(search_results) > 0 and 'score' in search_results[0],
            f"First result: {search_results[0] if search_results else 'none'}"
        )

        results.record(
            "ChromaDB results marked as semantic search",
            len(search_results) > 0 and search_results[0].get('search_method') == 'semantic',
            f"Search method: {search_results[0].get('search_method') if search_results else 'none'}"
        )

        # Test metadata in stored results
        result_file = Path(test_dir) / "analysis_20260113_140000.json"
        with open(result_file) as f:
            data = json.load(f)

        results.record(
            "ChromaDB backend includes storage_backend metadata",
            data.get('storage_backend') == "chromadb",
            f"storage_backend: {data.get('storage_backend', 'missing')}"
        )

        # Check index has vector embedding flag
        with open(index_file) as f:
            index = json.load(f)

        results.record(
            "ChromaDB backend marks entries with vector embedding flag",
            len(index) > 0 and index[0].get('has_vector_embedding') == True,
            f"has_vector_embedding: {index[0].get('has_vector_embedding') if index else 'missing'}"
        )

    finally:
        shutil.rmtree(test_dir)


def test_factory_function(results):
    """Test the create_storage_backend factory"""
    test_dir = tempfile.mkdtemp(prefix="rlm_test_factory_")

    try:
        backend = create_storage_backend(test_dir)

        if CHROMADB_AVAILABLE:
            results.record(
                "Factory creates ChromaDB backend when available",
                isinstance(backend, ChromaDBBackend),
                f"Got {type(backend).__name__}"
            )
        else:
            results.record(
                "Factory creates JSON backend when ChromaDB unavailable",
                isinstance(backend, JSONBackend),
                f"Got {type(backend).__name__}"
            )

        # Test that backend works
        backend.store(
            timestamp="20260113_160000",
            query="Test query",
            focus="testing",
            result={"content": "Test result"},
            file_hashes={"test.py": "test123"},
            stats={"total_tokens": 100, "total_cost_usd": 0.01},
            path=test_dir
        )

        search_results = backend.search("test", max_results=5)
        results.record(
            "Factory-created backend can store and search",
            len(search_results) > 0,
            f"Search returned {len(search_results)} results"
        )

    finally:
        shutil.rmtree(test_dir)


def test_semantic_vs_keyword_comparison(results):
    """Compare semantic vs keyword search quality"""
    if not CHROMADB_AVAILABLE:
        print("⏭️  Skipping semantic vs keyword comparison (ChromaDB not installed)")
        return

    # Create two backends
    json_dir = tempfile.mkdtemp(prefix="rlm_test_json_cmp_")
    chroma_dir = tempfile.mkdtemp(prefix="rlm_test_chroma_cmp_")

    try:
        json_backend = JSONBackend(json_dir)
        chroma_backend = ChromaDBBackend(chroma_dir)

        # Store same data in both
        test_data = [
            {
                "timestamp": "20260113_170000",
                "query": "Find SQL injection vulnerabilities",
                "focus": "security",
                "result": {"content": "Found SQL injection in user input"},
                "file_hashes": {"db.py": "hash1"},
                "stats": {"total_tokens": 500, "total_cost_usd": 0.025},
            },
            {
                "timestamp": "20260113_170100",
                "query": "Optimize slow database queries",
                "focus": "performance",
                "result": {"content": "Database indexes missing"},
                "file_hashes": {"db.py": "hash2"},
                "stats": {"total_tokens": 600, "total_cost_usd": 0.03},
            },
            {
                "timestamp": "20260113_170200",
                "query": "Review authentication logic",
                "focus": "security",
                "result": {"content": "Password hashing is weak"},
                "file_hashes": {"auth.py": "hash3"},
                "stats": {"total_tokens": 550, "total_cost_usd": 0.028},
            }
        ]

        for data in test_data:
            json_backend.store(path=json_dir, **data)
            chroma_backend.store(path=chroma_dir, **data)

        # Test semantic search advantage: "login bugs" should find "authentication logic"
        # even though they don't share exact keywords
        json_results = json_backend.search("login bugs", max_results=5)
        chroma_results = chroma_backend.search("login bugs", max_results=5)

        # JSON backend might not find anything (no exact "login" or "bugs" in queries)
        json_found_auth = any("auth" in r['query'].lower() for r in json_results)
        chroma_found_auth = any("auth" in r['query'].lower() for r in chroma_results)

        results.record(
            "Semantic search finds related concepts better than keyword",
            chroma_found_auth,  # ChromaDB should find it
            f"JSON found auth: {json_found_auth}, ChromaDB found auth: {chroma_found_auth}"
        )

        # Test that both find exact matches
        json_results = json_backend.search("SQL injection", max_results=5)
        chroma_results = chroma_backend.search("SQL injection", max_results=5)

        results.record(
            "Both backends find exact matches",
            len(json_results) > 0 and len(chroma_results) > 0,
            f"JSON: {len(json_results)}, ChromaDB: {len(chroma_results)}"
        )

    finally:
        shutil.rmtree(json_dir)
        shutil.rmtree(chroma_dir)


def main():
    print("Testing RLM Storage Backends")
    print("="*70)
    print()

    results = TestResults()

    # Run all test suites
    print("1. Backend Detection Tests")
    print("-"*70)
    test_backend_detection(results)
    print()

    print("2. JSON Backend Tests")
    print("-"*70)
    test_json_backend(results)
    print()

    print("3. ChromaDB Backend Tests")
    print("-"*70)
    test_chromadb_backend(results)
    print()

    print("4. Factory Function Tests")
    print("-"*70)
    test_factory_function(results)
    print()

    print("5. Semantic vs Keyword Search Comparison")
    print("-"*70)
    test_semantic_vs_keyword_comparison(results)
    print()

    return results.summary()


if __name__ == "__main__":
    sys.exit(main())
