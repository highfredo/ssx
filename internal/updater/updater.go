package updater

import (
	"context"
	"log/slog"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/highfredo/ssx/internal/appconfig"
)

const (
	githubRepo   = "highfredo/ssx"
	checksumFile = "checksums.txt"
)

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
		slog.Warn("no se pudo escribir sample.config.yaml", "err", err)
	}

	slog.Info("ssx updated", "version", latest.Version())
	return true, nil
}
