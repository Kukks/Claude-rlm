package main

import (
	"context"
	"fmt"
	"os"

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
	Version   = "dev"
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

func init() {
	rootCmd.AddCommand(mcpCmd, analyzeCmd, updateCmd, statusCmd)
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
		RAGDir:         cfg.Storage.RAGDir,
		QdrantAddress:  cfg.Storage.QdrantAddress,
		QdrantEnabled:  cfg.Storage.QdrantEnabled,
		CollectionName: cfg.Storage.CollectionName,
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
	fmt.Printf("  Qdrant Enabled: %v\n", cfg.Storage.QdrantEnabled)
	fmt.Printf("  Qdrant Address: %s\n", cfg.Storage.QdrantAddress)
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
