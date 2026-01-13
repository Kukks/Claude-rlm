#!/usr/bin/env python3
"""
Auto-detect environment and provide installation instructions for RLM MCP
"""
import os
import sys
import platform
import json
from pathlib import Path

def detect_environment():
    """Detect OS and Claude installation type"""
    os_type = platform.system()  # 'Windows', 'Darwin', 'Linux'

    # Check for Claude Desktop
    claude_desktop_configs = {
        'Darwin': Path.home() / 'Library/Application Support/Claude/claude_desktop_config.json',
        'Windows': Path(os.getenv('APPDATA', '')) / 'Claude/claude_desktop_config.json',
        'Linux': Path.home() / '.config/Claude/claude_desktop_config.json'
    }

    # Check for Claude Code CLI
    claude_code_configs = {
        'Darwin': Path.home() / '.config/claude/config.json',
        'Windows': Path(os.getenv('APPDATA', '')) / 'claude/config.json',
        'Linux': Path.home() / '.config/claude/config.json'
    }

    has_desktop = claude_desktop_configs.get(os_type, Path()).exists()
    has_code_cli = claude_code_configs.get(os_type, Path()).exists()

    return {
        'os': os_type,
        'os_name': {'Darwin': 'macOS', 'Windows': 'Windows', 'Linux': 'Linux'}.get(os_type, os_type),
        'claude_desktop': {
            'installed': has_desktop,
            'config_path': str(claude_desktop_configs.get(os_type, ''))
        },
        'claude_code': {
            'installed': has_code_cli,
            'config_path': str(claude_code_configs.get(os_type, ''))
        }
    }

def get_rlm_path():
    """Get absolute path to RLM server"""
    script_dir = Path(__file__).parent
    return script_dir / 'mcp_server' / 'rlm_server.py'

def generate_mcp_config(env):
    """Generate MCP configuration for detected environment"""
    rlm_path = str(get_rlm_path().resolve())

    config = {
        "mcpServers": {
            "rlm": {
                "command": "python3" if env['os'] != 'Windows' else "python",
                "args": [rlm_path],
                "env": {}
            }
        }
    }

    return config

def print_instructions(env):
    """Print installation instructions based on environment"""

    print("=" * 70)
    print("RLM MCP Server - Installation Instructions")
    print("=" * 70)
    print()

    print(f"📍 Detected Environment: {env['os_name']}")
    print()

    rlm_path = get_rlm_path().resolve()
    config = generate_mcp_config(env)

    # Check what's installed
    if env['claude_desktop']['installed']:
        print("✅ Claude Desktop detected")
        print(f"   Config: {env['claude_desktop']['config_path']}")
        print()
        print_desktop_instructions(env, config)

    if env['claude_code']['installed']:
        print("✅ Claude Code CLI detected")
        print(f"   Config: {env['claude_code']['config_path']}")
        print()
        print_code_cli_instructions(env, config)

    if not env['claude_desktop']['installed'] and not env['claude_code']['installed']:
        print("⚠️  No Claude installation detected")
        print()
        print("Please install one of:")
        print("  - Claude Desktop: https://claude.ai/download")
        print("  - Claude Code CLI: npm install -g @anthropic-ai/claude-code")
        print()
        print("Then run this script again.")

def print_desktop_instructions(env, config):
    """Print Claude Desktop installation instructions"""
    print("📦 Claude Desktop Installation")
    print("-" * 70)
    print()
    print("1. Open the config file:")
    print(f"   {env['claude_desktop']['config_path']}")
    print()

    if env['os'] == 'Darwin':
        print("   Quick open:")
        print(f"   open ~/Library/Application\\ Support/Claude/")
    elif env['os'] == 'Windows':
        print("   Quick open:")
        print(f"   explorer %APPDATA%\\Claude")
    else:  # Linux
        print("   Quick open:")
        print(f"   xdg-open ~/.config/Claude/")

    print()
    print("2. Add this configuration:")
    print()
    print(json.dumps(config, indent=2))
    print()
    print("3. Restart Claude Desktop completely")
    print()
    print("4. Verify installation:")
    print("   - Look for MCP indicator in Claude")
    print("   - Try: 'How does authentication work in this repo?'")
    print()

def print_code_cli_instructions(env, config):
    """Print Claude Code CLI installation instructions"""
    print("⚡ Claude Code CLI Installation")
    print("-" * 70)
    print()
    print("1. Open the config file:")
    print(f"   {env['claude_code']['config_path']}")
    print()

    if env['os'] == 'Darwin' or env['os'] == 'Linux':
        print("   Quick edit:")
        print(f"   nano ~/.config/claude/config.json")
    else:  # Windows
        print("   Quick edit:")
        print(f"   notepad %APPDATA%\\claude\\config.json")

    print()
    print("2. Add this to the 'mcpServers' section:")
    print()
    print(json.dumps(config, indent=2))
    print()
    print("3. Restart your terminal")
    print()
    print("4. Verify installation:")
    print("   claude mcp list")
    print("   # Should show 'rlm' server")
    print()
    print("5. Test it:")
    print("   cd your-project")
    print("   claude")
    print("   > How does authentication work here?")
    print()

def check_python_version():
    """Check if Python version is compatible"""
    version = sys.version_info
    if version.major < 3 or (version.major == 3 and version.minor < 8):
        print("⚠️  Warning: Python 3.8+ required")
        print(f"   Current version: {version.major}.{version.minor}.{version.micro}")
        print()
        return False
    else:
        print(f"✅ Python {version.major}.{version.minor}.{version.micro} (compatible)")
        print()
        return True

def check_chromadb():
    """Check if ChromaDB is installed for semantic search"""
    try:
        import chromadb
        print(f"✅ ChromaDB {chromadb.__version__} (semantic search enabled)")
        print()
        return True
    except ImportError:
        print("⚠️  ChromaDB not installed (semantic search disabled)")
        print("   Recommended: pip install chromadb")
        print("   Without it: Uses basic keyword search instead")
        print()
        return False

def test_server():
    """Test if RLM server can start"""
    rlm_path = get_rlm_path()

    if not rlm_path.exists():
        print(f"❌ Error: RLM server not found at {rlm_path}")
        return False

    print(f"✅ RLM server found at:")
    print(f"   {rlm_path}")
    print()

    return True

def main():
    print()

    # Check Python version
    if not check_python_version():
        print("Please upgrade Python: https://www.python.org/downloads/")
        sys.exit(1)

    # Check ChromaDB for semantic search
    has_chromadb = check_chromadb()

    # Test server exists
    if not test_server():
        sys.exit(1)

    # Detect environment
    env = detect_environment()

    # Print instructions
    print_instructions(env)

    print("=" * 70)
    print()

    if not has_chromadb:
        print("🚀 RECOMMENDED: Install ChromaDB for semantic search")
        print("   pip install chromadb")
        print()
        print("   Benefits:")
        print("   • Find 'auth bugs' → discovers 'login vulnerabilities'")
        print("   • Semantic understanding of code and queries")
        print("   • 10x better search quality than keyword matching")
        print()
        print("   Or install all optional dependencies:")
        print("   pip install -r requirements.txt")
        print()
        print("=" * 70)
        print()

    print("📚 Documentation:")
    print("   - Quick Start: QUICKSTART_MCP.md")
    print("   - Full Guide: MCP_INTEGRATION.md")
    print("   - Examples: EXAMPLE_PROMPTS.md")
    print("   - Staleness Detection: STALENESS_DETECTION.md")
    print()
    print("💡 Need help? https://github.com/Kukks/claude-rlm/issues")
    print()

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nInstallation cancelled.")
        sys.exit(0)
    except Exception as e:
        print(f"\n❌ Error: {e}")
        sys.exit(1)
