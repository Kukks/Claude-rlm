package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kukks/claude-rlm/internal/config"
	"github.com/kukks/claude-rlm/internal/mcp"
	"github.com/kukks/claude-rlm/internal/orchestrator"
	"github.com/kukks/claude-rlm/internal/storage"
	"github.com/kukks/claude-rlm/internal/updater"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	Version   = "3.0.3"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "rlm",
	Short: "Claude RLM - Recursive Language Models",
	Long: `Claude RLM implements the Recursive Language Models pattern for analyzing
documents beyond context window limits using intelligent decomposition and
trampoline-based recursion.`,
	Version: Version,
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run MCP server (called by Claude)",
	Long:  `Starts the MCP server on stdio for integration with Claude Desktop or Claude Code CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMCPServer(); err != nil {
			log.Fatal().Err(err).Msg("MCP server failed")
		}
	},
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path] [query]",
	Short: "Analyze a file or directory",
	Long:  `Directly analyze a file or directory with a given query (for testing).`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		query := args[1]

		if err := runAnalyze(path, query); err != nil {
			log.Fatal().Err(err).Msg("Analysis failed")
		}
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update to latest version",
	Long:  `Check for and install the latest version of RLM from GitHub releases.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runUpdate(); err != nil {
			log.Fatal().Err(err).Msg("Update failed")
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status and statistics",
	Long:  `Display current status, statistics, and configuration information.`,
	Run: func(cmd *cobra.Command, args []string) {
		showStatus()
	},
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install RLM and configure Claude",
	Long: `Install RLM binary to PATH and automatically configure Claude Desktop
and Claude Code CLI to use RLM as an MCP server.

This command will:
1. Copy the rlm binary to a directory in your PATH
2. Detect Claude Desktop configuration and add RLM
3. Detect Claude Code CLI configuration and add RLM`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInstall(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd, analyzeCmd, updateCmd, statusCmd, installCmd)
}

func setupLogger(cfg *config.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Set log level
	level := zerolog.InfoLevel
	switch cfg.Logging.Level {
	case "debug":
		level = zerolog.DebugLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	}

	// Set log format
	var logger zerolog.Logger
	if cfg.Logging.Format == "json" {
		logger = zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()
	} else {
		logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(level)
	}

	return logger
}

func runMCPServer() error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Setup logger
	logger := setupLogger(cfg)

	// Create orchestrator
	orchConfig := &orchestrator.Config{
		MaxRecursionDepth: cfg.Orchestrator.MaxRecursionDepth,
		MaxIterations:     cfg.Orchestrator.MaxIterations,
		CacheEnabled:      cfg.Orchestrator.CacheEnabled,
		CacheTTL:          cfg.Orchestrator.CacheTTL(),
		WorkDir:           ".",
	}

	orch := orchestrator.New(orchConfig, logger)
	orch.SetDispatcher(orchestrator.PlaceholderDispatcher)

	// Create storage backend
	storageConfig := &storage.Config{
		RAGDir: cfg.Storage.RAGDir,
	}

	backend, err := storage.NewBackend(ctx, storageConfig)
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}
	defer backend.Close()

	logger.Info().Str("storage", backend.Name()).Msg("Storage backend initialized")

	// Create MCP server
	server := mcp.NewServer(orch, backend, logger, Version)
	defer server.Close()

	// Check for updates on startup (non-blocking)
	if cfg.Updater.Enabled {
		go checkForUpdates(ctx, logger)
	}

	// Run MCP server
	return server.RunStdio(ctx)
}

func runAnalyze(path, query string) error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Setup logger
	logger := setupLogger(cfg)

	// Create orchestrator
	orchConfig := &orchestrator.Config{
		MaxRecursionDepth: cfg.Orchestrator.MaxRecursionDepth,
		MaxIterations:     cfg.Orchestrator.MaxIterations,
		CacheEnabled:      cfg.Orchestrator.CacheEnabled,
		CacheTTL:          cfg.Orchestrator.CacheTTL(),
		WorkDir:           ".",
	}

	orch := orchestrator.New(orchConfig, logger)
	orch.SetDispatcher(orchestrator.PlaceholderDispatcher)

	// Run analysis
	result, err := orch.AnalyzeDocument(ctx, path, query)
	if err != nil {
		return err
	}

	// Display result
	fmt.Println("Analysis Result:")
	fmt.Println("================")
	fmt.Println(result.Content)
	fmt.Println()
	fmt.Printf("Token Count: %d\n", result.TokenCount)
	fmt.Printf("Cost: $%.4f\n", result.CostUSD)

	stats := orch.GetStats()
	fmt.Println()
	fmt.Println("Statistics:")
	fmt.Printf("  Subagent Calls: %d\n", stats.TotalSubagentCalls)
	fmt.Printf("  Total Tokens: %d\n", stats.TotalTokens)
	fmt.Printf("  Total Cost: $%.4f\n", stats.TotalCostUSD)
	fmt.Printf("  Max Depth: %d\n", stats.MaxDepthReached)
	fmt.Printf("  Cache Hits: %d\n", stats.CacheHits)

	return nil
}

func runUpdate() error {
	ctx := context.Background()
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	upd := updater.New(Version, logger)

	fmt.Println("Checking for updates...")

	release, hasUpdate, err := upd.CheckForUpdate(ctx)
	if err != nil {
		return err
	}

	if !hasUpdate {
		fmt.Println("Already running the latest version:", Version)
		return nil
	}

	fmt.Printf("Update available: %s -> %s\n", Version, *release.TagName)
	fmt.Print("Install update? [Y/n]: ")

	var response string
	fmt.Scanln(&response)

	if response == "" || response == "y" || response == "Y" {
		fmt.Println("Downloading and installing update...")
		if err := upd.Update(ctx, release); err != nil {
			return err
		}
		fmt.Println("Update successful! Restart RLM to use the new version.")
	} else {
		fmt.Println("Update cancelled.")
	}

	return nil
}

func showStatus() {
	fmt.Printf("RLM Version: %s\n", Version)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Config: Using defaults (no config file found)")
	} else {
		fmt.Println("Config: Loaded from file")
	}

	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  Max Recursion Depth: %d\n", cfg.Orchestrator.MaxRecursionDepth)
	fmt.Printf("  Cache Enabled: %v\n", cfg.Orchestrator.CacheEnabled)
	fmt.Printf("  Storage Backend: BM25 (pure Go)\n")
	fmt.Printf("  RAG Directory: %s\n", cfg.Storage.RAGDir)
}

func checkForUpdates(ctx context.Context, logger zerolog.Logger) {
	upd := updater.New(Version, logger)

	release, hasUpdate, err := upd.CheckForUpdate(ctx)
	if err != nil {
		logger.Debug().Err(err).Msg("Update check failed")
		return
	}

	if hasUpdate {
		fmt.Fprintf(os.Stderr, "\n✨ Update available: %s\nRun 'rlm update' to install\n\n", *release.TagName)
	}
}

// runInstall handles the install command
func runInstall() error {
	fmt.Println("RLM Installer")
	fmt.Println("=============")
	fmt.Println()

	// Step 1: Install binary to PATH
	binaryPath, err := installBinary()
	if err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}
	fmt.Printf("✓ Binary installed to: %s\n", binaryPath)

	// Step 2: Configure Claude Desktop
	desktopConfigured, err := configureClaudeDesktop(binaryPath)
	if err != nil {
		fmt.Printf("⚠ Claude Desktop configuration failed: %v\n", err)
	} else if desktopConfigured {
		fmt.Println("✓ Claude Desktop configured")
	} else {
		fmt.Println("- Claude Desktop not found (skipped)")
	}

	// Step 3: Configure Claude Code CLI
	codeConfigured, err := configureClaudeCode(binaryPath)
	if err != nil {
		fmt.Printf("⚠ Claude Code configuration failed: %v\n", err)
	} else if codeConfigured {
		fmt.Println("✓ Claude Code CLI configured")
	} else {
		fmt.Println("- Claude Code CLI not found (skipped)")
	}

	fmt.Println()
	fmt.Println("Installation complete!")
	if desktopConfigured {
		fmt.Println("  • Restart Claude Desktop to use RLM")
	}
	if codeConfigured {
		fmt.Println("  • Restart your terminal to use RLM with Claude Code")
	}
	fmt.Println()
	fmt.Println("Test with: rlm status")

	return nil
}

// installBinary copies the current binary to a directory in PATH
func installBinary() (string, error) {
	// Get current executable path
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Determine target directory based on OS
	var targetDir string
	var targetName string

	switch runtime.GOOS {
	case "windows":
		// Use %LOCALAPPDATA%\Programs\rlm or %USERPROFILE%\bin
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			targetDir = filepath.Join(localAppData, "Programs", "rlm")
		} else {
			home, _ := os.UserHomeDir()
			targetDir = filepath.Join(home, "bin")
		}
		targetName = "rlm.exe"
	case "darwin", "linux":
		// Try /usr/local/bin first, fall back to ~/.local/bin
		if _, err := os.Stat("/usr/local/bin"); err == nil {
			// Check if we can write to it
			testFile := "/usr/local/bin/.rlm_test"
			if f, err := os.Create(testFile); err == nil {
				f.Close()
				os.Remove(testFile)
				targetDir = "/usr/local/bin"
			}
		}
		if targetDir == "" {
			home, _ := os.UserHomeDir()
			targetDir = filepath.Join(home, ".local", "bin")
		}
		targetName = "rlm"
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Create target directory if needed
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	targetPath := filepath.Join(targetDir, targetName)

	// Check if source and target are the same
	if exe == targetPath {
		return targetPath, nil
	}

	// Copy binary
	src, err := os.Open(exe)
	if err != nil {
		return "", fmt.Errorf("failed to open source: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create target: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy binary: %w", err)
	}

	// On Windows, add to PATH if needed
	if runtime.GOOS == "windows" {
		addToWindowsPath(targetDir)
	}

	return targetPath, nil
}

// addToWindowsPath adds a directory to the user's PATH on Windows
func addToWindowsPath(dir string) {
	// Get current user PATH
	path := os.Getenv("PATH")
	if strings.Contains(strings.ToLower(path), strings.ToLower(dir)) {
		return // Already in PATH
	}

	fmt.Printf("\nNote: Add this directory to your PATH: %s\n", dir)
	fmt.Println("You can do this by running (in PowerShell as Admin):")
	fmt.Printf("  [Environment]::SetEnvironmentVariable('PATH', $env:PATH + ';%s', 'User')\n", dir)
}

// configureClaudeDesktop adds RLM to Claude Desktop configuration
func configureClaudeDesktop(binaryPath string) (bool, error) {
	configPath := getClaudeDesktopConfigPath()
	if configPath == "" {
		return false, nil
	}

	// Check if config directory exists
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return false, nil // Claude Desktop not installed
	}

	return addMCPServerToConfig(configPath, binaryPath)
}

// configureClaudeCode adds RLM to Claude Code CLI configuration
func configureClaudeCode(binaryPath string) (bool, error) {
	configPath := getClaudeCodeConfigPath()
	if configPath == "" {
		return false, nil
	}

	// Check if config directory exists
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// Try to detect if Claude Code is installed by checking for the binary
		if !isClaudeCodeInstalled() {
			return false, nil
		}
		// Create config directory
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return false, err
		}
	}

	return addMCPServerToConfig(configPath, binaryPath)
}

// getClaudeDesktopConfigPath returns the Claude Desktop config path for the current OS
func getClaudeDesktopConfigPath() string {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "Claude", "claude_desktop_config.json")
		}
	case "linux":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	}
	return ""
}

// getClaudeCodeConfigPath returns the Claude Code CLI config path for the current OS
func getClaudeCodeConfigPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join(home, ".claude", "settings.json")
	case "windows":
		return filepath.Join(home, ".claude", "settings.json")
	}
	return ""
}

// isClaudeCodeInstalled checks if Claude Code CLI is installed
func isClaudeCodeInstalled() bool {
	// Check common locations
	paths := []string{"claude", "claude.exe"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}

	// Check in PATH
	pathEnv := os.Getenv("PATH")
	var separator string
	if runtime.GOOS == "windows" {
		separator = ";"
	} else {
		separator = ":"
	}

	for _, dir := range strings.Split(pathEnv, separator) {
		claudePath := filepath.Join(dir, "claude")
		if runtime.GOOS == "windows" {
			claudePath += ".exe"
		}
		if _, err := os.Stat(claudePath); err == nil {
			return true
		}
	}

	return false
}

// addMCPServerToConfig adds or updates the RLM MCP server in a config file
func addMCPServerToConfig(configPath, binaryPath string) (bool, error) {
	// Read existing config or create new one
	var config map[string]interface{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			config = make(map[string]interface{})
		} else {
			return false, err
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return false, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	// Get or create mcpServers section
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
		config["mcpServers"] = mcpServers
	}

	// Add RLM server
	mcpServers["rlm"] = map[string]interface{}{
		"command": binaryPath,
		"args":    []string{"mcp"},
	}

	// Write config back
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return false, err
	}

	return true, nil
}
