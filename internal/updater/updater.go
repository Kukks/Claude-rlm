package updater

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/inconshreveable/go-update"
	"github.com/rs/zerolog"
)

const (
	Owner = "Kukks"
	Repo  = "claude-rlm"
)

// Updater handles auto-updates
type Updater struct {
	client         *github.Client
	currentVersion string
	logger         zerolog.Logger
}

// New creates a new updater
func New(currentVersion string, logger zerolog.Logger) *Updater {
	return &Updater{
		client:         github.NewClient(nil),
		currentVersion: currentVersion,
		logger:         logger,
	}
}

// CheckForUpdate checks if a newer version is available
func (u *Updater) CheckForUpdate(ctx context.Context) (*github.RepositoryRelease, bool, error) {
	release, _, err := u.client.Repositories.GetLatestRelease(ctx, Owner, Repo)
	if err != nil {
		return nil, false, err
	}

	if release.TagName == nil {
		return nil, false, fmt.Errorf("no tag name in release")
	}

	latest := *release.TagName
	if latest > u.currentVersion {
		return release, true, nil
	}

	return release, false, nil
}

// Update downloads and applies an update
func (u *Updater) Update(ctx context.Context, release *github.RepositoryRelease) error {
	// Find asset for current platform
	assetName := fmt.Sprintf("rlm_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}

	var assetURL string
	for _, asset := range release.Assets {
		if asset.Name != nil && *asset.Name == assetName {
			assetURL = *asset.BrowserDownloadURL
			break
		}
	}

	if assetURL == "" {
		return fmt.Errorf("no asset found for platform %s_%s", runtime.GOOS, runtime.GOARCH)
	}

	u.logger.Info().Str("url", assetURL).Msg("Downloading update")

	// Download binary
	resp, err := http.Get(assetURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Apply update
	err = update.Apply(resp.Body, update.Options{})
	if err != nil {
		if rerr := update.RollbackError(err); rerr != nil {
			u.logger.Error().Err(rerr).Msg("Failed to rollback")
		}
		return err
	}

	u.logger.Info().Str("version", *release.TagName).Msg("Update successful")
	return nil
}

// AutoUpdate runs periodic update checks in the background
func (u *Updater) AutoUpdate(ctx context.Context, interval time.Duration, autoApply bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			release, hasUpdate, err := u.CheckForUpdate(ctx)
			if err != nil {
				u.logger.Warn().Err(err).Msg("Update check failed")
				continue
			}

			if hasUpdate {
				u.logger.Info().Str("version", *release.TagName).Msg("Update available")

				if autoApply {
					if err := u.Update(ctx, release); err != nil {
						u.logger.Error().Err(err).Msg("Auto-update failed")
					}
				}
			}

		case <-ctx.Done():
			return
		}
	}
}
