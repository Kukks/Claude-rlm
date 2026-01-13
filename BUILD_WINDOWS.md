# Building RLM on Windows

## Prerequisites

1. Install Go 1.23+: https://go.dev/dl/
2. Install Git for Windows: https://git-scm.com/download/win

## Build Steps

```powershell
# Clone the repository
git clone https://github.com/Kukks/claude-rlm.git
cd claude-rlm

# Checkout the branch
git checkout claude/build-rlm-orchestrator-dw3se

# Build for Windows
$env:CGO_ENABLED="0"
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -ldflags="-s -w" -o rlm.exe ./cmd/rlm

# Verify it works
.\rlm.exe --version
```

## Quick Build (One Command)

```powershell
$env:CGO_ENABLED="0"; go build -ldflags="-s -w" -o rlm.exe ./cmd/rlm
```

The `rlm.exe` will be in your current directory (~10MB).

## Install to System

```powershell
# Option 1: Move to your PATH
Move-Item rlm.exe "$env:USERPROFILE\bin\rlm.exe"

# Option 2: Use from current directory
# Just reference .\rlm.exe
```

## MCP Configuration

Edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "rlm": {
      "command": "C:\\Users\\YourUsername\\path\\to\\rlm.exe",
      "args": ["mcp"]
    }
  }
}
```

Replace `YourUsername` and path with your actual path.
