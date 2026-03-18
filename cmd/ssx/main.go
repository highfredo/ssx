// ssx is a terminal user interface for managing SSH connections and tunnels.
//
// It reads host configuration from ~/.ssh/config, presents an interactive
// list of hosts, and allows the user to:
//   - Connect interactively via SSH (key: enter or c)
//   - Open and close individual port-forwarding tunnels (key: t → tunnel view)
//
// All active tunnels are gracefully terminated when the program exits.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/paths"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/ui"
	"github.com/highfredo/ssx/internal/updater"
)

// from ldflags
var (
	version string
	date    string
	commit  string
)

func main() {
	// Print version
	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println("Version: " + version)
		fmt.Println("Date: " + date)
		fmt.Println("Commit: " + commit)
		return
	}

	setupLogger()
	slog.Info("ssx starting", "version", version)

	go func() {
		updater.CheckAndUpdate(version)
	}()

	// Servicios
	tunnelManager := ssh.NewTunnelManager()

	// App
	model := ui.NewApp(tunnelManager)
	p := tea.NewProgram(model)
	tunnelManager.SetEmitter(func() {
		p.Send(ssh.TunnelStateChangedMsg{})
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
	slog.Info("ssx exited cleanly")
}

func setupLogger() {
	_ = os.MkdirAll(paths.CacheDir(), 0o655)
	logPath := filepath.Join(paths.CacheDir(), "ssx.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: appconfig.Get().SlogLevel(),
	})))
}
