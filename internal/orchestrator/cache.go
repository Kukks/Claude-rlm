package orchestrator

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	CacheDir       = ".rlm_cache"
	DefaultCacheTTL = 24 * time.Hour
)

// CacheEntry represents a cached result with metadata
type CacheEntry struct {
	Result    *AnalysisResult `json:"result"`
	Timestamp time.Time       `json:"timestamp"`
	TTL       time.Duration   `json:"ttl"`
}

// GenerateCacheKey creates a SHA256 hash of the task parameters
func GenerateCacheKey(task *Task) string {
	// Sort context keys for consistent hashing
	contextKeys := make([]string, 0, len(task.Context))
	for k := range task.Context {
		contextKeys = append(contextKeys, k)
	}
	sort.Strings(contextKeys)

	// Build deterministic context string
	sortedContext := make(map[string]interface{})
	for _, k := range contextKeys {
		sortedContext[k] = task.Context[k]
	}

	contextJSON, _ := json.Marshal(sortedContext)

	// Hash: agent_type + task_description + sorted_context
	key := fmt.Sprintf("%s|%s|%s", task.AgentType, task.TaskDescription, string(contextJSON))
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// CheckCache looks for a valid cached result
func (o *Orchestrator) CheckCache(task *Task) *AnalysisResult {
	if !o.config.CacheEnabled {
		return nil
	}

	cacheKey := GenerateCacheKey(task)
	cacheFile := filepath.Join(o.config.WorkDir, CacheDir, cacheKey+".json")

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil // Cache miss
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		o.logger.Warn().Err(err).Str("cache_key", cacheKey).Msg("Failed to parse cache entry")
		return nil
	}

	// Check TTL
	if time.Since(entry.Timestamp) > entry.TTL {
		o.logger.Debug().Str("cache_key", cacheKey).Msg("Cache entry expired")
		os.Remove(cacheFile) // Clean up expired entry
		return nil
	}

	o.logger.Debug().Str("cache_key", cacheKey).Msg("Cache hit")
	return entry.Result
}

// StoreCache saves a result to the cache
func (o *Orchestrator) StoreCache(task *Task, result *AnalysisResult) error {
	if !o.config.CacheEnabled {
		return nil
	}

	cacheKey := GenerateCacheKey(task)
	cacheDir := filepath.Join(o.config.WorkDir, CacheDir)

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	entry := CacheEntry{
		Result:    result,
		Timestamp: time.Now(),
		TTL:       o.config.CacheTTL,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	cacheFile := filepath.Join(cacheDir, cacheKey+".json")
	return os.WriteFile(cacheFile, data, 0644)
}

// ClearCache removes all cache entries
func (o *Orchestrator) ClearCache() error {
	cacheDir := filepath.Join(o.config.WorkDir, CacheDir)
	return os.RemoveAll(cacheDir)
}

// CleanExpiredCache removes expired cache entries
func (o *Orchestrator) CleanExpiredCache() error {
	cacheDir := filepath.Join(o.config.WorkDir, CacheDir)

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cleaned := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		cacheFile := filepath.Join(cacheDir, entry.Name())
		data, err := os.ReadFile(cacheFile)
		if err != nil {
			continue
		}

		var cacheEntry CacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			continue
		}

		if time.Since(cacheEntry.Timestamp) > cacheEntry.TTL {
			os.Remove(cacheFile)
			cleaned++
		}
	}

	if cleaned > 0 {
		o.logger.Info().Int("cleaned", cleaned).Msg("Cleaned expired cache entries")
	}

	return nil
}
