package hash

import (
	"fmt"
	"time"
)

// StalenessReport contains information about file changes
type StalenessReport struct {
	Stale          bool      `json:"stale"`
	ChangedFiles   []string  `json:"changed_files"`
	NewFiles       []string  `json:"new_files"`
	DeletedFiles   []string  `json:"deleted_files"`
	TotalChanges   int       `json:"total_changes"`
	LastAnalysis   string    `json:"last_analysis"`
	Recommendation string    `json:"recommendation"`
	Details        string    `json:"details"`
}

// CheckStaleness compares stored file hashes with current hashes
func CheckStaleness(storedHashes map[string]string, currentPath string, lastAnalysisTime time.Time) (*StalenessReport, error) {
	// Compute current hashes
	hasher := NewFileHasher()
	currentHashes, err := hasher.ComputeDirectoryHash(currentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute current hashes: %w", err)
	}

	// Find changes
	changedFiles := FindChangedFiles(storedHashes, currentHashes)
	newFiles := FindNewFiles(storedHashes, currentHashes)
	deletedFiles := FindDeletedFiles(storedHashes, currentHashes)

	totalChanges := len(changedFiles) + len(newFiles) + len(deletedFiles)
	stale := totalChanges > 0

	// Generate report
	report := &StalenessReport{
		Stale:        stale,
		ChangedFiles: changedFiles,
		NewFiles:     newFiles,
		DeletedFiles: deletedFiles,
		TotalChanges: totalChanges,
		LastAnalysis: lastAnalysisTime.Format("2006-01-02 15:04:05"),
	}

	// Generate recommendation
	if !stale {
		report.Recommendation = "Analysis is up to date. No re-analysis needed."
		report.Details = "No file changes detected since last analysis."
	} else {
		report.Recommendation = fmt.Sprintf("Re-analyze recommended. %d file(s) changed.", totalChanges)
		report.Details = report.buildDetailsString()
	}

	return report, nil
}

// buildDetailsString creates a detailed description of changes
func (r *StalenessReport) buildDetailsString() string {
	details := ""

	if len(r.ChangedFiles) > 0 {
		details += fmt.Sprintf("Modified: %d file(s)\n", len(r.ChangedFiles))
		for i, f := range r.ChangedFiles {
			if i < 5 {
				details += fmt.Sprintf("  - %s\n", f)
			} else {
				details += fmt.Sprintf("  ... and %d more\n", len(r.ChangedFiles)-5)
				break
			}
		}
	}

	if len(r.NewFiles) > 0 {
		details += fmt.Sprintf("Added: %d file(s)\n", len(r.NewFiles))
		for i, f := range r.NewFiles {
			if i < 5 {
				details += fmt.Sprintf("  - %s\n", f)
			} else {
				details += fmt.Sprintf("  ... and %d more\n", len(r.NewFiles)-5)
				break
			}
		}
	}

	if len(r.DeletedFiles) > 0 {
		details += fmt.Sprintf("Deleted: %d file(s)\n", len(r.DeletedFiles))
		for i, f := range r.DeletedFiles {
			if i < 5 {
				details += fmt.Sprintf("  - %s\n", f)
			} else {
				details += fmt.Sprintf("  ... and %d more\n", len(r.DeletedFiles)-5)
				break
			}
		}
	}

	return details
}

// String returns a human-readable staleness report
func (r *StalenessReport) String() string {
	if !r.Stale {
		return "✓ Analysis is fresh - no changes detected"
	}

	return fmt.Sprintf("⚠ Analysis is stale - %d change(s) detected\n%s", r.TotalChanges, r.Details)
}

// IsCritical checks if the changes are significant enough to require immediate re-analysis
func (r *StalenessReport) IsCritical() bool {
	// Consider critical if:
	// - More than 10% of files changed
	// - More than 50 files changed
	// - Any core files changed (could be customized per project)

	if r.TotalChanges > 50 {
		return true
	}

	// This is a simplification - in real usage you might want to check
	// specific important files or directories
	return false
}

// GetSummary returns a one-line summary
func (r *StalenessReport) GetSummary() string {
	if !r.Stale {
		return "Fresh"
	}

	return fmt.Sprintf("%d changed, %d new, %d deleted",
		len(r.ChangedFiles), len(r.NewFiles), len(r.DeletedFiles))
}
