// ssx is a terminal user interface for managing SSH connections and tunnels.
//
// It reads host configuration from ~/.ssh/config, presents an interactive
// list of hosts, and allows the user to:
//   - Connect interactively via SSH (key: enter or c)
//   - Open and close individual port-forwarding tunnels (key: t → tunnel view)
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/cli"
	"github.com/highfredo/ssx/internal/paths"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/updater"
)

// Injected via ldflags at build time.
var (
	version string
	date    string
	commit  string
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := appconfig.Load()
	setupLogger(cfg.SlogLevel())
	slog.Info("ssx starting", "version", version)

	handled, err := cli.Parse(cli.BuildInfo{Version: version, Date: date, Commit: commit})
	if handled || err != nil {
		return err
	}

	styles.Apply(cfg.Theme)
	keys := base.NewKeybindings(cfg.Keybindings)

	hosts, err := ssh.LoadConfig()
	if err != nil {
		return err
	}
	p := tea.NewProgram(ui.NewApp(hosts, cfg, keys))

	go updater.CheckAndUpdate(version)
	// Notify the user if a previous background update installed a new version.
	if v := updater.ConsumePendingUpdate(); v != "" {
		go p.Send(base.AppUpdatedMsg{NewVersion: v})
	}

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	slog.Info("ssx exited cleanly")
	return nil
}

// setupLogger sends slog output to $XDG_CACHE_HOME/ssx/ssx.log. Logging is
// best-effort: when the file cannot be opened the default handler is kept.
func setupLogger(level slog.Level) {
	if err := os.MkdirAll(paths.CacheDir(), 0o755); err != nil {
		return
	}
	logPath := filepath.Join(paths.CacheDir(), "ssx.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o664)
	if err != nil {
		return
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: level,
	})))
}
