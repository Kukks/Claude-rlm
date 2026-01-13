#!/usr/bin/env python3
"""
Auto-detect environment and provide installation instructions for RLM MCP
"""
import os
import sys
import platform
import json
import subprocess
import shutil
from pathlib import Path

def detect_wsl_installations():
    """Detect WSL installations (when running on Windows)"""
    if platform.system() != 'Windows':
        return []

    wsl_installs = []

    try:
        # Get list of WSL distributions
        result = subprocess.run(
            ['wsl', '--list', '--quiet'],
            capture_output=True,
            text=True,
            encoding='utf-16-le',  # WSL outputs UTF-16-LE
            timeout=5
        )

        if result.returncode != 0:
            return []

        # Parse distributions (filter out empty lines)
        distros = [line.strip().replace('\x00', '') for line in result.stdout.split('\n') if line.strip()]

        for distro in distros:
            if not distro:
                continue

            # Check if Claude Code CLI exists in this WSL distro
            check_cmd = ['wsl', '-d', distro, '--', 'bash', '-c',
                        'test -f ~/.config/claude/config.json && echo "exists"']

            try:
                check_result = subprocess.run(
                    check_cmd,
                    capture_output=True,
                    text=True,
                    timeout=3
                )

                if 'exists' in check_result.stdout:
                    # Get the Windows path to the WSL config
                    wsl_path_cmd = ['wsl', '-d', distro, '--', 'bash', '-c',
                                   'wslpath -w ~/.config/claude/config.json']
                    path_result = subprocess.run(
                        wsl_path_cmd,
                        capture_output=True,
                        text=True,
                        timeout=3
                    )

                    if path_result.returncode == 0:
                        win_path = path_result.stdout.strip()
                        wsl_installs.append({
                            'distro': distro,
                            'config_path': win_path,
                            'linux_path': '~/.config/claude/config.json'
                        })
            except (subprocess.TimeoutExpired, Exception):
                continue

    except (FileNotFoundError, subprocess.TimeoutExpired, Exception):
        # WSL not available or error occurred
        pass

    return wsl_installs

def detect_environment():
    """Detect OS and Claude installation types (including WSL)"""
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

    # Detect WSL installations
    wsl_installations = detect_wsl_installations() if os_type == 'Windows' else []

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
        },
        'wsl_installations': wsl_installations
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

def print_installation_table(env):
    """Print a table of all detected installations"""
    installations = []

    # Claude Desktop
    if env['claude_desktop']['installed']:
        installations.append({
            'name': 'Claude Desktop',
            'type': 'desktop',
            'status': '✅ Found',
            'config': env['claude_desktop']['config_path']
        })

    # Claude Code CLI (native)
    if env['claude_code']['installed']:
        platform_name = env['os_name']
        installations.append({
            'name': f'Claude Code ({platform_name})',
            'type': 'code_cli',
            'status': '✅ Found',
            'config': env['claude_code']['config_path']
        })

    # WSL installations
    for wsl in env.get('wsl_installations', []):
        installations.append({
            'name': f'Claude Code (WSL: {wsl["distro"]})',
            'type': 'wsl',
            'status': '✅ Found',
            'config': wsl['config_path'],
            'linux_path': wsl['linux_path']
        })

    if not installations:
        return None

    print("🔍 Detected Installations:")
    print()
    print("┌─" + "─" * 30 + "┬─" + "─" * 10 + "┬─" + "─" * 50 + "┐")
    print("│ " + "Installation".ljust(30) + "│ " + "Status".ljust(10) + "│ " + "Config Path".ljust(50) + "│")
    print("├─" + "─" * 30 + "┼─" + "─" * 10 + "┼─" + "─" * 50 + "┤")

    for inst in installations:
        # Truncate long paths for table display
        config_display = inst['config']
        if len(config_display) > 50:
            config_display = "..." + config_display[-47:]

        print("│ " + inst['name'].ljust(30) + "│ " + inst['status'].ljust(10) + "│ " + config_display.ljust(50) + "│")

    print("└─" + "─" * 30 + "┴─" + "─" * 10 + "┴─" + "─" * 50 + "┘")
    print()

    return installations

def update_config_file(config_path, mcp_config):
    """Update a config file with RLM MCP server configuration"""
    config_path = Path(config_path)

    # Create parent directory if needed
    config_path.parent.mkdir(parents=True, exist_ok=True)

    # Read existing config or create new one
    if config_path.exists():
        with open(config_path, 'r') as f:
            try:
                existing = json.load(f)
            except json.JSONDecodeError:
                existing = {}
    else:
        existing = {}

    # Add or update mcpServers section
    if 'mcpServers' not in existing:
        existing['mcpServers'] = {}

    existing['mcpServers']['rlm'] = mcp_config['mcpServers']['rlm']

    # Write back
    with open(config_path, 'w') as f:
        json.dump(existing, f, indent=2)

    return True

def auto_configure(env, installations):
    """Automatically configure all detected installations"""
    config = generate_mcp_config(env)

    print("🔧 Auto-Configuration")
    print("-" * 70)
    print()

    success_count = 0
    fail_count = 0

    for inst in installations:
        name = inst['name']
        config_path = inst['config']

        try:
            # For WSL, we need to write using WSL commands
            if inst['type'] == 'wsl':
                # Extract distro name
                distro = inst['name'].split('WSL: ')[1].rstrip(')')

                # Create a simpler approach: write config via Python in WSL
                rlm_config_json = json.dumps(config['mcpServers']['rlm'])

                # Build command without f-string backslashes
                bash_script = (
                    'mkdir -p ~/.config/claude && '
                    'python3 -c "'
                    'import json, os; '
                    'path = os.path.expanduser(\'~/.config/claude/config.json\'); '
                    'cfg = json.load(open(path)) if os.path.exists(path) else {}; '
                    'cfg.setdefault(\'mcpServers\', {}); '
                    'cfg[\'mcpServers\'][\'rlm\'] = ' + rlm_config_json.replace('"', '\\"') + '; '
                    'json.dump(cfg, open(path, \'w\'), indent=2)'
                    '"'
                )

                wsl_cmd = ['wsl', '-d', distro, '--', 'bash', '-c', bash_script]
                result = subprocess.run(wsl_cmd, capture_output=True, timeout=5)

                if result.returncode == 0:
                    print(f"  ✅ {name}")
                    success_count += 1
                else:
                    print(f"  ❌ {name} (error writing config)")
                    fail_count += 1
            else:
                # Native Windows/Mac/Linux
                update_config_file(config_path, config)
                print(f"  ✅ {name}")
                success_count += 1

        except Exception as e:
            print(f"  ❌ {name} ({str(e)[:50]}...)")
            fail_count += 1

    print()
    print(f"Results: {success_count} configured, {fail_count} failed")
    print()

    if success_count > 0:
        print("✅ Configuration complete!")
        print()
        print("Next steps:")
        if env['claude_desktop']['installed']:
            print("  • Restart Claude Desktop completely")
        if env['claude_code']['installed'] or env.get('wsl_installations'):
            print("  • Restart your terminal/WSL session")
        print()

    return success_count > 0

def print_instructions(env):
    """Print installation instructions based on environment"""

    print("=" * 70)
    print("RLM MCP Server - Installation Instructions")
    print("=" * 70)
    print()

    print(f"📍 Detected Environment: {env['os_name']}")
    print()

    # Print installation table
    installations = print_installation_table(env)

    if not installations:
        print("⚠️  No Claude installation detected")
        print()
        print("Please install one of:")
        print("  - Claude Desktop: https://claude.ai/download")
        print("  - Claude Code CLI: npm install -g @anthropic-ai/claude-code")
        print()
        print("Then run this script again.")
        return

    # Offer auto-configuration
    print("Would you like to auto-configure all detected installations?")
    print("(This will add RLM MCP server to each config file)")
    print()

    try:
        response = input("Configure all? [Y/n]: ").strip().lower()
    except (EOFError, KeyboardInterrupt):
        response = 'n'
        print()

    if response in ['', 'y', 'yes']:
        print()
        if auto_configure(env, installations):
            return  # Success, no need for manual instructions

    # Manual instructions fallback
    print()
    print("📝 Manual Configuration Instructions:")
    print("-" * 70)
    print()

    rlm_path = get_rlm_path().resolve()
    config = generate_mcp_config(env)

    # Check what's installed
    if env['claude_desktop']['installed']:
        print_desktop_instructions(env, config)

    if env['claude_code']['installed']:
        print_code_cli_instructions(env, config)

    # WSL instructions
    for wsl in env.get('wsl_installations', []):
        print_wsl_instructions(env, config, wsl)

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

def print_wsl_instructions(env, config, wsl):
    """Print WSL installation instructions"""
    print(f"🐧 WSL Installation ({wsl['distro']})")
    print("-" * 70)
    print()
    print("From Windows PowerShell/CMD:")
    print()
    print(f"1. Access WSL ({wsl['distro']}):")
    print(f"   wsl -d {wsl['distro']}")
    print()
    print("2. Edit the config file:")
    print(f"   nano {wsl['linux_path']}")
    print()
    print("3. Add this to the 'mcpServers' section:")
    print()
    print(json.dumps(config, indent=2))
    print()
    print("4. Exit WSL and restart your WSL terminal")
    print()
    print("5. Test it from WSL:")
    print(f"   wsl -d {wsl['distro']}")
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
