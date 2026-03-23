package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

type TunnelStateChangedMsg struct {
	ID           string
	TunnelStatus TunnelStatus
	PortStatus   PortStatus
}

type TunnelState int

const (
	TunnelClosed TunnelState = iota
	TunnelOpenning
	TunnelOpened
)

type TunnelStatus struct {
	State TunnelState
	PID   int
	Err   error
}

type TunnelManager struct {
	store *tunnelStatusStore
}

func NewTunnelManager() *TunnelManager {
	tm := &TunnelManager{
		store: newTunnelStatusStore(),
	}
	tm.store.load()
	return tm
}

func (m *TunnelManager) SetEmitter(send func(any)) {
	m.store.onChange = func(id string, e tunnelEntry) {
		send(TunnelStateChangedMsg{
			ID:           id,
			TunnelStatus: e.tunnelStatus,
			PortStatus:   e.portStatus,
		})
	}
}

func (m *TunnelManager) Open(tunnel Tunnel) {
	id := TunnelID(tunnel)

	// Mark as opening — clears any previous error.
	m.store.SetPID(id, -1)

	h := GetHost(tunnel.Host)

	// Build Args
	args := []string{
		"-v",                             // required to detect when forwarding is ready
		"-N",                             // No remote command — stay alive for port forwarding only.
		"-o", "ExitOnForwardFailure=yes", // Die immediately if forwarding fails.
		"-F", "/dev/null", // Ignore host-level forwarding directives.
		// Keep each tunnel in its own SSH process. This prevents ControlMaster
		// multiplexing from collapsing multiple tunnel lifecycles into one.
		"-o", "ControlMaster=no",
		"-o", "ServerAliveInterval=30", // Detect broken connections.
		"-o", "ServerAliveCountMax=3",
		"-o", "StrictHostKeyChecking=accept-new", // never prompt TTY for new keys
	}

	// Tunnels run unattended without a TTY. If no password is configured,
	// SSH must never hang waiting for interactive input — fail fast instead.
	if h.Password == "" && h.PasswordCommand == "" {
		args = append(args, "-o", "BatchMode=yes")
	}
	switch tunnel.Type {
	case TunnelTypeLocal:
		args = append(args, "-L", tunnel.LocalPort+":"+tunnel.RemoteHost+":"+tunnel.RemotePort)
	case TunnelTypeRemote:
		args = append(args, "-R", tunnel.LocalPort+":"+tunnel.RemoteHost+":"+tunnel.RemotePort)
	case TunnelTypeDynamic:
		args = append(args, "-D", tunnel.LocalPort)
	}

	slog.Info("Opening", "tunnel", tunnel)
	cmd := PrepareCmd(h, args...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("Error getting stderr pipe", "err", err)
		m.store.SetClosed(id, err)
		return
	}

	if err := cmd.Start(); err != nil {
		slog.Error("Error starting tunnel", "err", err)
		m.store.SetClosed(id, err)
		return
	}

	// Read stderr to detect when forwarding is established (event-driven, no polling).
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			slog.Debug("ssh", "id", id, "stderr", line)
			if isForwardingReady(tunnel.Type, line) {
				slog.Info("tunnel forwarding ready", "id", id, "pid", cmd.Process.Pid)
				cmd.CleanFn()
				m.store.SetPID(id, cmd.Process.Pid)
			}
		}
	}()

	// Wait for the process to exit and update state accordingly.
	go func() {
		err := cmd.Wait()
		if err != nil {
			if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr.ExitCode() < 0 {
				// Exit code < 0 means the process was killed by a signal
				// (e.g. from Close()). Expected — not a real error.
				err = nil
			} else {
				slog.Error("tunnel exited unexpectedly", "id", id, "error", err)
			}
		}
		m.store.SetClosed(id, err)
	}()
}

// OpenTunnels returns all tunnels currently in TunnelOpened state across all hosts.
func (m *TunnelManager) OpenTunnels(hostconfigs []*HostConfig) []Tunnel {
	var result []Tunnel
	for _, host := range hostconfigs {
		for _, t := range host.Tunnels {
			status, _ := m.Status(t)
			if status.State == TunnelOpened {
				result = append(result, t)
			}
		}
	}
	return result
}

func (m *TunnelManager) Close(tunnel Tunnel) error {
	id := TunnelID(tunnel)
	slog.Info("closing tunnel", "id", id)
	err := KillPortOwner(tunnel.LocalPort)
	m.store.SetClosed(id, nil)
	return err
}

func (m *TunnelManager) Status(tunnel Tunnel) (TunnelStatus, PortStatus) {
	entry := m.store.Get(TunnelID(tunnel))
	return entry.tunnelStatus, entry.portStatus
}

func (m *TunnelManager) RefreshStatus(tunnel Tunnel) {
	id := TunnelID(tunnel)
	ps, err := GetPortStatus(tunnel.LocalPort)
	if err != nil {
		slog.Warn("RefreshStatus: cannot get port status", "port", tunnel.LocalPort, "err", err)
		return
	}
	m.store.SetPortStatus(id, ps)
}

/*
**************

	Persistence

**************
*/

// isForwardingReady detects the stderr line that confirms SSH is listening
// on the local port.
func isForwardingReady(tunnelType TunnelType, line string) bool {
	switch tunnelType {
	case TunnelTypeLocal, TunnelTypeDynamic:
		return strings.Contains(line, "Local forwarding listening on")
	case TunnelTypeRemote:
		return strings.Contains(line, "remote forward success")
	}
	return false
}

/*
**************

	Utils

**************
*/

// TunnelID returns the stable unique key for a tunnel, scoped to its host alias.
// Exported so callers can match TunnelStateChangedMsg.ID against known tunnels.
func TunnelID(t Tunnel) string {
	return fmt.Sprintf("%s|%d|%s|%s|%s",
		t.Host, t.Type, t.LocalPort, t.RemoteHost, t.RemotePort)
}
