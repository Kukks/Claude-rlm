package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileHasher computes SHA256 hashes of files
type FileHasher struct {
	excludeDirs []string
	patterns    []string
}

// NewFileHasher creates a new file hasher with default patterns
func NewFileHasher() *FileHasher {
	return &FileHasher{
		excludeDirs: []string{".git", "node_modules", ".rlm", ".rlm_cache", "vendor", "dist", "build"},
		patterns: []string{
			"*.py", "*.js", "*.ts", "*.tsx", "*.jsx",
			"*.go", "*.rs", "*.java", "*.c", "*.cpp", "*.h",
			"*.md", "*.txt", "*.json", "*.yaml", "*.yml",
			"*.html", "*.css", "*.scss", "*.sass",
		},
	}
}

// ComputeFileHash returns the SHA256 hash of a single file
func ComputeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ComputeDirectoryHash returns a map of file paths to their SHA256 hashes
func (h *FileHasher) ComputeDirectoryHash(dirPath string) (map[string]string, error) {
	hashes := make(map[string]string)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Check if this directory should be excluded
			dirName := filepath.Base(path)
			for _, excluded := range h.excludeDirs {
				if dirName == excluded {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file matches any pattern
		if !h.matchesPattern(path) {
			return nil
		}

		// Compute hash
		hash, err := ComputeFileHash(path)
		if err != nil {
			// Skip files that can't be read
			return nil
		}

		// Store with relative path
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			relPath = path
		}

		hashes[relPath] = hash
		return nil
	})

	return hashes, err
}

// matchesPattern checks if a file matches any of the configured patterns
func (h *FileHasher) matchesPattern(filePath string) bool {
	fileName := filepath.Base(filePath)

	for _, pattern := range h.patterns {
		matched, err := filepath.Match(pattern, fileName)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// ComputeQuickHash computes a quick hash for a subset of files (for performance)
func (h *FileHasher) ComputeQuickHash(dirPath string, maxFiles int) (map[string]string, error) {
	hashes := make(map[string]string)
	count := 0

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if count >= maxFiles {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			dirName := filepath.Base(path)
			for _, excluded := range h.excludeDirs {
				if dirName == excluded {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file matches any pattern
		if !h.matchesPattern(path) {
			return nil
		}

		// Compute hash
		hash, err := ComputeFileHash(path)
		if err != nil {
			return nil
		}

		// Store with relative path
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			relPath = path
		}

		hashes[relPath] = hash
		count++

		return nil
	})

	return hashes, err
}

// SetPatterns allows customizing file patterns to hash
func (h *FileHasher) SetPatterns(patterns []string) {
	h.patterns = patterns
}

// SetExcludeDirs allows customizing directories to exclude
func (h *FileHasher) SetExcludeDirs(dirs []string) {
	h.excludeDirs = dirs
}

// HashesEqual compares two hash maps
func HashesEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}

// FindChangedFiles finds files that have changed between two hash maps
func FindChangedFiles(old, new map[string]string) []string {
	changed := make([]string, 0)

	for path, newHash := range new {
		oldHash, exists := old[path]
		if !exists || oldHash != newHash {
			changed = append(changed, path)
		}
	}

	return changed
}

// FindNewFiles finds files that exist in new but not in old
func FindNewFiles(old, new map[string]string) []string {
	newFiles := make([]string, 0)

	for path := range new {
		if _, exists := old[path]; !exists {
			newFiles = append(newFiles, path)
		}
	}

	return newFiles
}

// FindDeletedFiles finds files that exist in old but not in new
func FindDeletedFiles(old, new map[string]string) []string {
	deleted := make([]string, 0)

	for path := range old {
		if _, exists := new[path]; !exists {
			deleted = append(deleted, path)
		}
	}

	return deleted
}

// FormatHash truncates a hash for display
func FormatHash(hash string) string {
	if len(hash) > 12 {
		return hash[:12] + "..."
	}
	return hash
}

// IsTextFile checks if a file is likely a text file based on extension
func IsTextFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	textExts := []string{
		".txt", ".md", ".json", ".yaml", ".yml", ".xml",
		".py", ".js", ".ts", ".go", ".rs", ".java",
		".c", ".cpp", ".h", ".hpp", ".sh", ".bash",
		".html", ".css", ".scss", ".sass",
	}

	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}

	return false
}
