package updater

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/paths"
)

const (
	githubRepo   = "highfredo/ssx"
	checksumFile = "checksums.txt"
	newVerFile   = "new_ver.txt"
)

func pendingUpdateFile() string {
	return filepath.Join(paths.CacheDir(), newVerFile)
}

func ConsumePendingUpdate() string {
	data, err := os.ReadFile(pendingUpdateFile())
	if err != nil {
		return ""
	}
	_ = os.Remove(pendingUpdateFile())
	return strings.TrimSpace(string(data))
}

func CheckAndUpdate(currentVersion string) (bool, error) {
	if _, err := semver.NewVersion(currentVersion); err != nil {
		slog.Debug("update check skipped: invalid version", "version", currentVersion)
		return false, nil
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: checksumFile},
	})

	if err != nil {
		slog.Warn("update check error", "err", err)
		return false, err
	}

	latest, found, err := updater.DetectLatest(
		context.Background(),
		selfupdate.ParseSlug(githubRepo),
	)

	if err != nil {
		slog.Warn("update check: error connecting GitHub", "err", err)
		return false, err
	}

	if !found || latest.LessOrEqual(currentVersion) {
		return false, nil
	}
	slog.Info("update available", "version", latest.Version())

	exe, err := os.Executable()
	if err != nil {
		return false, err
	}

	if err := updater.UpdateTo(context.Background(), latest, exe); err != nil {
		return false, err
	}

	if err := appconfig.WriteSampleConfig(); err != nil {
		slog.Warn("could not write sample.config.yaml", "err", err)
	}

	newVersion := latest.Version()
	slog.Info("ssx updated", "version", newVersion)

	// Persist the new version so the next launch can show a notification.
	if err := os.WriteFile(pendingUpdateFile(), []byte(newVersion), 0o600); err != nil {
		slog.Warn("could not write pending update marker", "err", err)
	}

	return true, nil
}
