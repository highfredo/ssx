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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/highfredo/ssx/internal/sshconfig"
	"github.com/highfredo/ssx/internal/tunnel"
	"github.com/highfredo/ssx/internal/ui"
)

func main() {
	setupLogger()
	slog.Info("ssx starting")

	hosts, err := sshconfig.ParseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading ~/.ssh/config: %v\n", err)
		os.Exit(1)
	}
	if len(hosts) == 0 {
		fmt.Fprintln(os.Stderr, "no hosts found in ~/.ssh/config")
		os.Exit(0)
	}

	mgr := tunnel.NewManager()
	if err := mgr.Discover(hosts); err != nil {
		slog.Warn("failed to discover existing tunnel processes", "err", err)
	}
	model := ui.NewApp(hosts, mgr)

	p := tea.NewProgram(model, tea.WithAltScreen())

	// Wire the tunnel manager's dispatch callback to the Bubble Tea runtime so
	// unexpected tunnel exits are surfaced as TUI messages.
	mgr.SetDispatch(func(msg any) { p.Send(msg) })

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
	slog.Info("ssx exited cleanly")
}

// setupLogger redirects structured logs to /tmp/ssx.log so they don't
// interfere with the TUI output. The log file can be tailed for debugging:
//
//	tail -f /tmp/ssx.log
func setupLogger() {
	f, err := os.OpenFile("/tmp/ssx.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		// Non-fatal: proceed without file logging.
		return
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}
