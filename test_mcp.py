#!/usr/bin/env python3
"""
Test script for RLM MCP server
"""
import json
import sys
import os

# Add parent directory to path
sys.path.insert(0, os.path.dirname(__file__))

from mcp_server.rlm_server import MCPServer

def test_tools_list():
    """Test that tools are defined correctly"""
    server = MCPServer()

    print("✓ MCP Server initialized")
    print(f"✓ Found {len(server.tools)} tools:")

    for tool in server.tools:
        print(f"  - {tool['name']}: {tool['description'][:50]}...")

    return True

def test_tool_schemas():
    """Test that tool schemas are valid"""
    server = MCPServer()

    for tool in server.tools:
        schema = tool.get('inputSchema')
        if not schema:
            print(f"✗ Tool {tool['name']} missing inputSchema")
            return False

        if schema.get('type') != 'object':
            print(f"✗ Tool {tool['name']} schema type is not 'object'")
            return False

    print("✓ All tool schemas valid")
    return True

def test_rag_storage():
    """Test RAG storage functionality"""
    server = MCPServer()

    # Create test RAG directory
    test_dir = "test_rag"
    os.makedirs(test_dir, exist_ok=True)

    # Test storage
    file_hashes = {"test.txt": "abc123"}
    server._store_rag_data(
        path=test_dir,
        query="Test query",
        result={"content": "Test result"},
        focus="testing",
        file_hashes=file_hashes
    )

    # Check files created
    rlm_dir = os.path.join(test_dir, ".rlm")
    if not os.path.exists(rlm_dir):
        print("✗ .rlm directory not created")
        return False

    index_file = os.path.join(rlm_dir, "index.json")
    if not os.path.exists(index_file):
        print("✗ index.json not created")
        return False

    # Load and verify index
    with open(index_file) as f:
        index = json.load(f)

    if len(index) == 0:
        print("✗ Index is empty")
        return False

    print("✓ RAG storage working")

    # Cleanup
    import shutil
    shutil.rmtree(test_dir)

    return True

def test_rag_search():
    """Test RAG search functionality"""
    server = MCPServer()

    # Create test RAG data
    test_dir = "test_rag_search"
    os.makedirs(test_dir, exist_ok=True)

    # Store some test data
    file_hashes = {"test.py": "def123"}
    server._store_rag_data(
        path=test_dir,
        query="Find security vulnerabilities",
        result={"content": "Found 3 SQL injection risks"},
        focus="security",
        file_hashes=file_hashes
    )

    server._store_rag_data(
        path=test_dir,
        query="Analyze performance",
        result={"content": "Database queries are slow"},
        focus="performance",
        file_hashes=file_hashes
    )

    # Search for security-related items
    rlm_dir = os.path.join(test_dir, ".rlm")
    results = server._search_rag_directory(rlm_dir, "security", max_results=5)

    if len(results) == 0:
        print("✗ RAG search found no results")
        return False

    if "security" not in results[0]["query"].lower():
        print("✗ RAG search returned wrong result")
        return False

    print("✓ RAG search working")

    # Cleanup
    import shutil
    shutil.rmtree(test_dir)

    return True

if __name__ == "__main__":
    print("Testing RLM MCP Server...")
    print()

    tests = [
        ("Tools list", test_tools_list),
        ("Tool schemas", test_tool_schemas),
        ("RAG storage", test_rag_storage),
        ("RAG search", test_rag_search)
    ]

    passed = 0
    failed = 0

    for name, test_func in tests:
        print(f"Testing {name}...")
        try:
            if test_func():
                passed += 1
            else:
                failed += 1
                print(f"✗ {name} failed")
        except Exception as e:
            failed += 1
            print(f"✗ {name} failed with exception: {e}")
        print()

    print("="*50)
    print(f"Results: {passed} passed, {failed} failed")

    if failed == 0:
        print("✅ All tests passed!")
        sys.exit(0)
    else:
        print("❌ Some tests failed")
        sys.exit(1)
