// Package tunnel manages the lifecycle of SSH port-forwarding processes.
// Each tunnel maps to a background "ssh -N" process. The Manager interface
// provides Open/Close operations and tracks tunnel state (OPEN/CLOSED).
// When a process exits unexpectedly a callback is used to notify the TUI.
package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/highfredo/ssx/internal/sshconfig"
	"github.com/highfredo/ssx/internal/sshpass"
)

// State represents whether a tunnel process is currently running.
type State string

const (
	StateOpen   State = "OPEN"
	StateClosed State = "CLOSED"
)

// ExitedMsg is dispatched via the registered dispatch callback when a tunnel
// exits unexpectedly (i.e. not via an explicit Close call).
type ExitedMsg struct {
	TunnelID string
	Err      error
}

// Manager defines the operations for managing SSH tunnel processes.
type Manager interface {
	// Open starts a background SSH port-forwarding process for the tunnel.
	// Returns an error if the tunnel is already open.
	Open(hostName string, t sshconfig.Tunnel, password string) error

	// Close terminates the tunnel process identified by tunnelID.
	Close(tunnelID string) error

	// State returns whether a tunnel is currently active.
	State(tunnelID string) State

	// Discover scans running ssh processes and marks configured tunnels as OPEN
	// when matching forwarding processes are already active.
	Discover(hosts []*sshconfig.Host) error

	// CloseAll terminates every active tunnel. Intended for graceful shutdown.
	CloseAll() error

	// SetDispatch registers a callback used to send out-of-band messages
	// (e.g. unexpected tunnel exits) back to the Bubble Tea runtime.
	SetDispatch(fn func(msg any))
}

// activeTunnel holds the running SSH process and its cancellation function.
type activeTunnel struct {
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	pid     int
	cleanup func()
}

// manager is the concrete Manager implementation.
type manager struct {
	mu        sync.RWMutex
	active    map[string]*activeTunnel
	dispatch  func(msg any)
	statePath string
}

// NewManager creates a ready-to-use Manager with no active tunnels.
func NewManager() Manager {
	return &manager{
		active:    make(map[string]*activeTunnel),
		statePath: defaultStatePath(),
	}
}

// SetDispatch registers the Bubble Tea send callback.
func (m *manager) SetDispatch(fn func(msg any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dispatch = fn
}

// Open starts an SSH port-forwarding process for the given host + tunnel pair.
// The ssh binary is invoked with -N (no remote command) so the process stays
// alive only for forwarding, and with keep-alive options to detect broken links.
func (m *manager) Open(hostName string, t sshconfig.Tunnel, password string) error {
	id := t.ID(hostName)

	m.mu.Lock()
	if _, exists := m.active[id]; exists {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %q is already open", id)
	}
	m.mu.Unlock()

	args := buildSSHArgs(hostName, t)
	slog.Info("opening tunnel", "id", id, "args", args)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "ssh", args...) //nolint:gosec
	cleanup, err := sshpass.Configure(cmd, password)
	if err != nil {
		cancel()
		return fmt.Errorf("configure password auth for tunnel %q: %w", id, err)
	}
	if err := cmd.Start(); err != nil {
		cleanup()
		cancel()
		return fmt.Errorf("start ssh for tunnel %q: %w", id, err)
	}

	m.mu.Lock()
	m.active[id] = &activeTunnel{cmd: cmd, cancel: cancel, pid: cmd.Process.Pid, cleanup: cleanup}
	if err := m.persistLocked(); err != nil {
		slog.Warn("persist tunnel state", "err", err)
	}
	m.mu.Unlock()

	// Background goroutine: wait for exit and notify if unexpected.
	go m.monitorExit(id)

	return nil
}

// Close cancels the context of a running tunnel, causing the SSH process to exit.
func (m *manager) Close(tunnelID string) error {
	m.mu.Lock()
	at, exists := m.active[tunnelID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %q is not open", tunnelID)
	}
	delete(m.active, tunnelID)
	if err := m.persistLocked(); err != nil {
		slog.Warn("persist tunnel state", "err", err)
	}
	m.mu.Unlock()

	slog.Info("closing tunnel", "id", tunnelID)
	if at.cleanup != nil {
		at.cleanup()
	}
	if at.cancel != nil {
		at.cancel()
		return nil
	}
	if at.pid <= 0 {
		return fmt.Errorf("tunnel %q has no process handle", tunnelID)
	}
	if err := syscall.Kill(at.pid, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
		return fmt.Errorf("terminate tunnel %q (pid %d): %w", tunnelID, at.pid, err)
	}
	return nil
}

// State returns the current state of a tunnel by ID.
func (m *manager) State(tunnelID string) State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.active[tunnelID]; ok {
		return StateOpen
	}
	return StateClosed
}

// Discover restores OPEN state for app-managed tunnels from a persistent state
// file and verifies each saved PID is still alive.
func (m *manager) Discover(hosts []*sshconfig.Host) error {
	saved, err := readPersistedState(m.statePath)
	if err != nil {
		return err
	}

	allowed := make(map[string]struct{})
	for _, h := range hosts {
		for _, t := range h.Tunnels {
			allowed[t.ID(h.Name)] = struct{}{}
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	restored := 0
	for id, pid := range saved {
		if _, ok := allowed[id]; !ok {
			continue
		}
		if pid > 0 && isProcessAlive(pid) {
			m.active[id] = &activeTunnel{pid: pid}
			restored++
		}
	}
	if err := m.persistLocked(); err != nil {
		slog.Warn("persist tunnel state", "err", err)
	}
	slog.Info("tunnel discovery complete", "restored", restored, "saved", len(saved))
	return nil
}

// CloseAll terminates all active tunnels and is safe to call multiple times.
func (m *manager) CloseAll() error {
	m.mu.Lock()
	ids := make([]string, 0, len(m.active))
	for id := range m.active {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	var lastErr error
	for _, id := range ids {
		if err := m.Close(id); err != nil {
			slog.Error("close tunnel on shutdown", "id", id, "err", err)
			lastErr = err
		}
	}
	return lastErr
}

// monitorExit waits for the tunnel process to exit. If the tunnel was not
// explicitly closed it dispatches an ExitedMsg to the TUI.
func (m *manager) monitorExit(id string) {
	m.mu.RLock()
	at, ok := m.active[id]
	m.mu.RUnlock()
	if !ok {
		return
	}

	err := at.cmd.Wait()
	if at.cleanup != nil {
		at.cleanup()
	}

	m.mu.Lock()
	_, stillActive := m.active[id]
	if stillActive {
		// Process died on its own — remove it and notify.
		delete(m.active, id)
		if err := m.persistLocked(); err != nil {
			slog.Warn("persist tunnel state", "err", err)
		}
	}
	dispatch := m.dispatch
	m.mu.Unlock()

	if stillActive {
		slog.Warn("tunnel exited unexpectedly", "id", id, "err", err)
		if dispatch != nil {
			dispatch(ExitedMsg{TunnelID: id, Err: err})
		}
	}
}

// buildSSHArgs constructs the argument list for an "ssh -N" tunnel command.
// Using the host alias lets SSH resolve HostName, User, IdentityFile, etc.
// from the user's own ~/.ssh/config.
func buildSSHArgs(hostName string, t sshconfig.Tunnel) []string {
	args := []string{
		"-N",                             // No remote command — stay alive for port forwarding only.
		"-o", "ExitOnForwardFailure=yes", // Die immediately if forwarding fails.
		"-o", "ServerAliveInterval=30", // Detect broken connections.
		"-o", "ServerAliveCountMax=3",
	}

	switch t.Type {
	case sshconfig.TunnelTypeLocal:
		args = append(args, "-L",
			fmt.Sprintf("%s:%s:%s", t.LocalPort, t.RemoteHost, t.RemotePort))
	case sshconfig.TunnelTypeRemote:
		args = append(args, "-R",
			fmt.Sprintf("%s:%s:%s", t.LocalPort, t.RemoteHost, t.RemotePort))
	case sshconfig.TunnelTypeDynamic:
		args = append(args, "-D", t.LocalPort)
	}

	return append(args, hostName)
}

func defaultStatePath() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "ssx", "tunnels.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "tunnels.json"
	}
	return filepath.Join(home, ".local", "state", "ssx", "tunnels.json")
}

func readPersistedState(path string) (map[string]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]int{}, nil
		}
		return nil, fmt.Errorf("read tunnel state %s: %w", path, err)
	}
	state := map[string]int{}
	if len(raw) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("decode tunnel state %s: %w", path, err)
	}
	return state, nil
}

func (m *manager) persistLocked() error {
	state := make(map[string]int, len(m.active))
	for id, at := range m.active {
		if at.pid > 0 {
			state[id] = at.pid
		}
	}
	dir := filepath.Dir(m.statePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create state dir %s: %w", dir, err)
	}
	b, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	tmp := m.statePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write state tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, m.statePath); err != nil {
		return fmt.Errorf("commit state file %s: %w", m.statePath, err)
	}
	return nil
}

func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
