# Quick Start - MCP Integration

Get RLM working with Claude in 5 minutes!

## Step 1: Clone Repository

```bash
git clone https://github.com/Kukks/claude-rlm.git
```

Note the absolute path (you'll need it).

## Step 2: Add to Claude Config

**macOS:**
```bash
nano ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

**Windows:**
```cmd
notepad %APPDATA%\Claude\claude_desktop_config.json
```

**Linux:**
```bash
nano ~/.config/Claude/claude_desktop_config.json
```

Add this (replace path!):

```json
{
  "mcpServers": {
    "rlm": {
      "command": "python3",
      "args": [
        "/FULL/PATH/TO/claude-rlm/mcp_server/rlm_server.py"
      ]
    }
  }
}
```

**Example (macOS):**
```json
{
  "mcpServers": {
    "rlm": {
      "command": "python3",
      "args": [
        "/Users/alice/projects/claude-rlm/mcp_server/rlm_server.py"
      ]
    }
  }
}
```

## Step 3: Restart Claude

Completely quit and restart Claude Desktop.

## Step 4: Test It!

In Claude, ask:

```
"Analyze this codebase - what are the main components?"
```

You should see Claude use the `rlm_analyze` tool automatically!

## Step 5: Check RAG Storage

In your project directory:

```bash
ls -la .rlm/
```

You should see:
```
.rlm/
├── index.json
└── analysis_20260113_140532.json
```

## That's It!

Now just ask Claude questions naturally:

- "How does authentication work here?"
- "Find security vulnerabilities"
- "What needs documentation?"
- "Explain the data flow"

RLM triggers automatically and stores results in `.rlm/` for instant retrieval later!

## Troubleshooting

### "Tool not found"

1. Check absolute path in config is correct
2. Verify Python 3.8+: `python3 --version`
3. Restart Claude Desktop

### "Permission denied"

```bash
chmod +x /path/to/claude-rlm/mcp_server/rlm_server.py
```

### Still not working?

Test manually:
```bash
cd claude-rlm
python3 mcp_server/rlm_server.py
```

Should show: "RLM MCP Server starting on stdio..."

## Next Steps

- [See 50+ example prompts](EXAMPLE_PROMPTS.md)
- [Read full MCP guide](MCP_INTEGRATION.md)
- [Learn RLM patterns](skills/rlm-patterns/SKILL.md)
