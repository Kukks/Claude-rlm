"""
Storage backend for RLM RAG data

Supports:
1. ChromaDB (Recommended) - Semantic vector search
2. JSON (Fallback) - Simple keyword search
"""
import os
import sys
import json
import hashlib
from typing import Dict, Any, List, Optional
from datetime import datetime
from pathlib import Path

# Try to import ChromaDB
try:
    import chromadb
    from chromadb.config import Settings
    CHROMADB_AVAILABLE = True
except ImportError:
    CHROMADB_AVAILABLE = False


class StorageBackend:
    """Abstract base for storage backends"""

    def __init__(self, rag_dir: str):
        self.rag_dir = Path(rag_dir)
        self.rag_dir.mkdir(parents=True, exist_ok=True)

    def store(self, timestamp: str, query: str, focus: str, result: Any,
              file_hashes: Dict[str, str], stats: Dict[str, Any], path: str):
        """Store analysis result"""
        raise NotImplementedError

    def search(self, query: str, max_results: int = 5) -> List[Dict]:
        """Search for relevant analyses"""
        raise NotImplementedError

    def get_all(self) -> List[Dict]:
        """Get all stored analyses"""
        raise NotImplementedError


class ChromaDBBackend(StorageBackend):
    """Vector database backend using ChromaDB for semantic search"""

    def __init__(self, rag_dir: str):
        super().__init__(rag_dir)

        # Initialize ChromaDB
        self.client = chromadb.PersistentClient(
            path=str(self.rag_dir / "chromadb"),
            settings=Settings(anonymized_telemetry=False)
        )

        # Get or create collection
        self.collection = self.client.get_or_create_collection(
            name="rlm_analyses",
            metadata={"description": "RLM analysis results with semantic search"}
        )

        # Also maintain JSON index for backwards compatibility and metadata
        self.index_file = self.rag_dir / "index.json"
        self.index = self._load_index()

    def _load_index(self) -> List[Dict]:
        """Load JSON index"""
        if self.index_file.exists():
            with open(self.index_file) as f:
                return json.load(f)
        return []

    def _save_index(self):
        """Save JSON index"""
        with open(self.index_file, 'w') as f:
            json.dump(self.index, f, indent=2)

    def store(self, timestamp: str, query: str, focus: str, result: Any,
              file_hashes: Dict[str, str], stats: Dict[str, Any], path: str):
        """Store analysis with vector embedding"""

        # Extract text content for embedding
        if isinstance(result, dict):
            content_text = result.get('content', str(result))
        else:
            content_text = str(result)

        # Prepare metadata
        metadata = {
            "timestamp": timestamp,
            "query": query,
            "focus": focus,
            "path": path,
            "files_tracked": len(file_hashes),
            "tokens": stats.get("total_tokens", 0),
            "cost_usd": stats.get("total_cost_usd", 0.0)
        }

        # Store in ChromaDB with embedding
        self.collection.add(
            documents=[content_text],
            metadatas=[metadata],
            ids=[timestamp]
        )

        # Store full data in JSON file for complete retrieval
        result_file = self.rag_dir / f"analysis_{timestamp}.json"
        with open(result_file, 'w') as f:
            json.dump({
                "query": query,
                "focus": focus,
                "timestamp": timestamp,
                "result": result if isinstance(result, dict) else {"content": str(result)},
                "stats": stats,
                "path": path,
                "file_hashes": file_hashes,
                "version": "2.0",
                "storage_backend": "chromadb"
            }, f, indent=2)

        # Update index
        self.index.append({
            "timestamp": timestamp,
            "query": query,
            "focus": focus,
            "file": f"analysis_{timestamp}.json",
            "path": path,
            "files_tracked": len(file_hashes),
            "has_hash_tracking": True,
            "has_vector_embedding": True
        })
        self._save_index()

    def search(self, query: str, max_results: int = 5) -> List[Dict]:
        """Semantic search using vector embeddings"""

        # Query ChromaDB with semantic search
        results = self.collection.query(
            query_texts=[query],
            n_results=max_results
        )

        # Load full data from JSON files
        formatted_results = []
        if results['ids'] and results['ids'][0]:
            for idx, doc_id in enumerate(results['ids'][0]):
                result_file = self.rag_dir / f"analysis_{doc_id}.json"
                if result_file.exists():
                    with open(result_file) as f:
                        full_data = json.load(f)

                    # Calculate relevance score (distance from ChromaDB)
                    distance = results['distances'][0][idx] if 'distances' in results else 0
                    score = 100 - (distance * 10)  # Convert distance to score

                    formatted_results.append({
                        "score": max(0, score),
                        "timestamp": doc_id,
                        "query": full_data.get("query"),
                        "focus": full_data.get("focus"),
                        "result": full_data.get("result"),
                        "stats": full_data.get("stats"),
                        "has_hash_tracking": True,
                        "has_vector_embedding": True,
                        "search_method": "semantic"
                    })

        return formatted_results

    def get_all(self) -> List[Dict]:
        """Get all analyses from index"""
        return self.index


class JSONBackend(StorageBackend):
    """Simple JSON-based storage with keyword search (fallback)"""

    def __init__(self, rag_dir: str):
        super().__init__(rag_dir)
        self.index_file = self.rag_dir / "index.json"
        self.index = self._load_index()

    def _load_index(self) -> List[Dict]:
        """Load JSON index"""
        if self.index_file.exists():
            with open(self.index_file) as f:
                return json.load(f)
        return []

    def _save_index(self):
        """Save JSON index"""
        with open(self.index_file, 'w') as f:
            json.dump(self.index, f, indent=2)

    def store(self, timestamp: str, query: str, focus: str, result: Any,
              file_hashes: Dict[str, str], stats: Dict[str, Any], path: str):
        """Store analysis in JSON"""

        # Store result file
        result_file = self.rag_dir / f"analysis_{timestamp}.json"
        with open(result_file, 'w') as f:
            json.dump({
                "query": query,
                "focus": focus,
                "timestamp": timestamp,
                "result": result if isinstance(result, dict) else {"content": str(result)},
                "stats": stats,
                "path": path,
                "file_hashes": file_hashes,
                "version": "2.0",
                "storage_backend": "json"
            }, f, indent=2)

        # Update index
        self.index.append({
            "timestamp": timestamp,
            "query": query,
            "focus": focus,
            "file": f"analysis_{timestamp}.json",
            "path": path,
            "files_tracked": len(file_hashes),
            "has_hash_tracking": True,
            "has_vector_embedding": False
        })
        self._save_index()

    def search(self, query: str, max_results: int = 5) -> List[Dict]:
        """Simple keyword search"""

        query_lower = query.lower()
        query_words = set(query_lower.split())

        scored_results = []
        for entry in self.index:
            entry_query = entry.get("query", "").lower()
            entry_focus = entry.get("focus", "").lower()

            # Calculate relevance score
            score = 0
            if query_lower in entry_query:
                score += 10

            entry_words = set(entry_query.split())
            common_words = query_words & entry_words
            score += len(common_words) * 2

            if query_lower in entry_focus:
                score += 5

            if score > 0:
                # Load full result
                result_file = self.rag_dir / entry["file"]
                if result_file.exists():
                    with open(result_file) as f:
                        full_data = json.load(f)

                    scored_results.append({
                        "score": score,
                        "timestamp": entry["timestamp"],
                        "query": entry["query"],
                        "focus": entry["focus"],
                        "result": full_data.get("result"),
                        "stats": full_data.get("stats"),
                        "has_hash_tracking": entry.get("has_hash_tracking", False),
                        "has_vector_embedding": False,
                        "search_method": "keyword"
                    })

        # Sort by score and return top results
        scored_results.sort(key=lambda x: x["score"], reverse=True)
        return scored_results[:max_results]

    def get_all(self) -> List[Dict]:
        """Get all analyses from index"""
        return self.index


def create_storage_backend(rag_dir: str) -> StorageBackend:
    """Factory function to create appropriate storage backend"""

    if CHROMADB_AVAILABLE:
        try:
            return ChromaDBBackend(rag_dir)
        except Exception as e:
            print(f"ChromaDB initialization failed: {e}", file=sys.stderr)
            print(f"Falling back to JSON storage", file=sys.stderr)
            return JSONBackend(rag_dir)
    else:
        return JSONBackend(rag_dir)


def get_backend_info() -> Dict[str, Any]:
    """Get information about available backends"""
    return {
        "chromadb_available": CHROMADB_AVAILABLE,
        "default_backend": "chromadb" if CHROMADB_AVAILABLE else "json",
        "semantic_search": CHROMADB_AVAILABLE
    }
