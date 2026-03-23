package ssh

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/highfredo/ssx/internal/atomicfile"
	"github.com/highfredo/ssx/internal/paths"
)

// tunnelEntry holds the full in-memory state for a single tunnel:
// process status (state, PID, last error) and the last known port status.
type tunnelEntry struct {
	tunnelStatus TunnelStatus
	portStatus   PortStatus
}

// tunnelStatusStore is the central in-memory state store for all tunnels.
// All operations are safe for concurrent use.
// Persistence is self-contained: every state change is automatically written
// to disk (only PIDs), and load() restores live tunnels on startup.
type tunnelStatusStore struct {
	mu       sync.RWMutex
	entries  map[string]tunnelEntry
	onChange func(id string, e tunnelEntry) // UI notification callback, set via TunnelManager.SetEmitter
}

func newTunnelStatusStore() *tunnelStatusStore {
	return &tunnelStatusStore{
		entries: make(map[string]tunnelEntry),
	}
}

// SetPID records the PID and derives TunnelState:
//
//	pid ≤ 0  →  TunnelOpenning
//	pid > 0  →  TunnelOpened
//
// Any previous error is cleared so a re-open always starts clean.
func (s *tunnelStatusStore) SetPID(id string, pid int) {
	state := TunnelOpenning
	if pid > 0 {
		state = TunnelOpened
	}
	s.mu.Lock()
	e := s.entries[id]
	e.tunnelStatus.State = state
	e.tunnelStatus.PID = pid
	e.tunnelStatus.Err = nil
	s.entries[id] = e
	s.mu.Unlock()
	s.notify(id, e)
}

// SetClosed marks the tunnel as closed, recording the exit error (may be nil).
// Passing nil clears any previous error.
// The cached port status is reset so stale data is never shown after closing.
func (s *tunnelStatusStore) SetClosed(id string, err error) {
	s.mu.Lock()
	e := s.entries[id]
	e.tunnelStatus.State = TunnelClosed
	e.tunnelStatus.PID = 0
	e.tunnelStatus.Err = err
	e.portStatus = PortStatus{} // invalidate — will be refreshed on next RefreshStatus
	s.entries[id] = e
	s.mu.Unlock()
	s.notify(id, e)
}

// SetPortStatus caches the latest port status for a tunnel.
// Only notifies when the value actually changes.
func (s *tunnelStatusStore) SetPortStatus(id string, ps PortStatus) {
	s.mu.Lock()
	e := s.entries[id]
	changed := e.portStatus != ps
	e.portStatus = ps
	s.entries[id] = e
	s.mu.Unlock()
	if changed {
		s.notify(id, e)
	}
}

// Get returns the current entry for the given tunnel ID.
// If the tunnel has never been registered the zero-value entry is returned
// (TunnelClosed state, PortClosed state, no error) — callers need not
// distinguish "missing" from "closed".
func (s *tunnelStatusStore) Get(id string) tunnelEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entries[id]
}

// Snapshot returns a map of tunnel ID → PID for disk persistence.
// Only tunnels that are fully open (PID > 0) are included.
func (s *tunnelStatusStore) Snapshot() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int)
	for id, e := range s.entries {
		if e.tunnelStatus.PID > 0 {
			out[id] = e.tunnelStatus.PID
		}
	}
	return out
}

func (s *tunnelStatusStore) notify(id string, e tunnelEntry) {
	s.save()
	if s.onChange != nil {
		s.onChange(id, e)
	}
}

// save persists confirmed tunnels (PID > 0) to disk.
func (s *tunnelStatusStore) save() {
	if err := atomicfile.WriteJSON(statePath(), s.Snapshot()); err != nil {
		slog.Error("failed to save tunnel state", "err", err)
	}
}

// load restores tunnels whose process is still alive and still owns the local
// port. Writes directly to s.entries to avoid triggering notify/save on data
// that was just read from disk. Must be called before SetEmitter.
func (s *tunnelStatusStore) load() {
	var state map[string]int
	if err := atomicfile.ReadJSON(statePath(), &state); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Error("failed to load tunnel state", "err", err)
		}
		return
	}

	for id, pid := range state {
		// tunnelID format: "host|type|localPort|remoteHost|remotePort"
		parts := strings.SplitN(id, "|", 5)
		if len(parts) < 3 {
			slog.Warn("ignoring malformed tunnel id in state", "id", id)
			continue
		}
		localPort := parts[2]

		portStatus, err := GetPortStatus(localPort)
		if err != nil {
			slog.Warn("could not check port on restore", "port", localPort, "err", err)
			continue
		}

		if portStatus.State == PortOpened && portStatus.PID == pid {
			slog.Info("restoring tunnel from persisted state", "id", id, "pid", pid)
			s.entries[id] = tunnelEntry{
				tunnelStatus: TunnelStatus{State: TunnelOpened, PID: pid},
				portStatus:   portStatus,
			}
		} else {
			slog.Info("discarding stale tunnel state", "id", id, "saved_pid", pid, "port_pid", portStatus.PID)
		}
	}
}

// statePath returns the path to the tunnel state file.
// Default: ~/.local/share/ssx/tunnels.json
func statePath() string {
	return filepath.Join(paths.DataDir(), "tunnels.json")
}
