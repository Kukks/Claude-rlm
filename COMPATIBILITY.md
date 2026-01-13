# Cross-Platform Compatibility

## Supported Platforms ✅

The RLM plugin is fully compatible with:

- **Windows** (Windows 10, 11, Server 2019+)
- **macOS** (10.15 Catalina and newer)
- **Linux** (Ubuntu 20.04+, Debian, Fedora, Arch, etc.)

## Python Requirements

- **Python 3.8+** (uses standard library only, no external dependencies)
- Works with system Python or virtual environments

## Cross-Platform Features

### ✅ File Paths
- Uses `os.path.join()` for all path concatenation
- Automatically uses correct separator (`/` on Unix, `\` on Windows)
- Example: `os.path.join("analysis", "chunk_1.json")` works everywhere

### ✅ Home Directory
- Uses `os.path.expanduser("~/.claude-rlm/config.json")`
- Resolves to:
  - Linux/Mac: `/home/user/.claude-rlm/config.json`
  - Windows: `C:\Users\username\.claude-rlm\config.json`

### ✅ Directory Creation
- Uses `os.makedirs(exist_ok=True)`
- Creates nested directories on all platforms
- Handles existing directories gracefully

### ✅ File Names
- All file names are cross-platform compatible:
  - `.rlm_state.json` (hidden on Unix, visible on Windows)
  - `analysis/` directory
  - `.rlm_cache/` directory (hidden on Unix)

### ✅ Text Encoding
- Python 3.8+ defaults to UTF-8
- JSON files use UTF-8 encoding
- Handles international characters correctly

### ✅ Line Endings
- Python automatically handles line endings:
  - `\n` on Unix/Mac
  - `\r\n` on Windows
- JSON output is consistent across platforms

## Usage Examples

### Windows
```cmd
# CMD
python src\orchestrator.py document.txt "Analyze this"

# PowerShell
python src\orchestrator.py document.txt "Analyze this"

# Git Bash
python src/orchestrator.py document.txt "Analyze this"
```

### macOS / Linux
```bash
# Make executable (optional)
chmod +x src/orchestrator.py

# Run directly
./src/orchestrator.py document.txt "Analyze this"

# Or with python
python3 src/orchestrator.py document.txt "Analyze this"
```

## Testing on Your Platform

Run the compatibility test:

```bash
# Test basic functionality
python src/orchestrator.py test_document.txt "Test analysis"

# Check status
python src/orchestrator.py status

# Verify paths
python -c "import os; print(os.path.expanduser('~/.claude-rlm/config.json'))"
```

Expected output:
- Windows: `C:\Users\<username>\.claude-rlm\config.json`
- Mac: `/Users/<username>/.claude-rlm/config.json`
- Linux: `/home/<username>/.claude-rlm/config.json`

## Known Platform Differences

### File Visibility
- **Unix/Mac**: Files starting with `.` are hidden by default
  - Use `ls -la` to see `.rlm_state.json` and `.rlm_cache/`
- **Windows**: All files are visible by default
  - `.rlm_state.json` shows in File Explorer

### Executable Permissions
- **Unix/Mac**: Need `chmod +x` to run `./orchestrator.py` directly
- **Windows**: Always use `python orchestrator.py`

### Shell Differences
- **Windows CMD**: Use backslashes or quotes for paths with spaces
- **PowerShell**: Similar to Unix shells
- **Git Bash on Windows**: Can use Unix-style forward slashes

## Path Examples

### Configuration File
```python
# Cross-platform (recommended)
config_path = os.path.expanduser("~/.claude-rlm/config.json")

# Results:
# Windows: C:\Users\Alice\.claude-rlm\config.json
# Mac:     /Users/Alice/.claude-rlm/config.json
# Linux:   /home/alice/.claude-rlm/config.json
```

### Cache Directory
```python
# Cross-platform (recommended)
cache_dir = ".rlm_cache"
cache_file = os.path.join(cache_dir, "abc123.json")

# Results:
# Windows: .rlm_cache\abc123.json
# Unix:    .rlm_cache/abc123.json
```

## Troubleshooting

### "Permission denied" on Unix/Mac
```bash
# Make orchestrator executable
chmod +x src/orchestrator.py

# Or always use python3 explicitly
python3 src/orchestrator.py
```

### "Python not found" on Windows
```cmd
# Use py launcher
py src\orchestrator.py status

# Or add Python to PATH
```

### Path issues with spaces
```bash
# Always quote paths with spaces
python src/orchestrator.py "My Documents/file.txt" "query"
```

## Developer Notes

When contributing, ensure cross-platform compatibility:

1. ✅ Use `os.path.join()` for paths
2. ✅ Use `os.path.expanduser()` for home directory
3. ✅ Use `os.makedirs(exist_ok=True)` for directories
4. ❌ Don't hardcode `/` or `\` in paths
5. ❌ Don't use Unix-specific commands (unless in optional hooks)
6. ❌ Don't assume case-sensitive file systems

## Automated Testing

The plugin includes platform detection:

```python
import platform

print(f"Platform: {platform.system()}")
# Windows: Windows
# Mac:     Darwin
# Linux:   Linux

print(f"Python: {platform.python_version()}")
# Ensure 3.8+
```

## Summary

**The RLM plugin is 100% cross-platform compatible** using only Python standard library. It works identically on Windows, macOS, and Linux without any modifications or platform-specific code.

The only differences are:
- File visibility (hidden files on Unix)
- Path separators (handled automatically)
- Shell syntax (use appropriate syntax for your shell)

All core functionality—trampoline logic, state persistence, caching, and analysis—works identically across all platforms.
