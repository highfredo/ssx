package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/highfredo/ssx/internal/atomicfile"
	"github.com/highfredo/ssx/internal/paths"
)

type TunnelStateChangedMsg struct {
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
	mapNotifier *MapNotifier
}

func NewTunnelManager() *TunnelManager {
	tm := &TunnelManager{
		mapNotifier: NewMapNotifier(),
	}
	tm.loadState()
	return tm
}

func (m *TunnelManager) SetEmitter(send func()) {
	m.mapNotifier.onChange = func() {
		m.saveState()
		send()
	}
}

func (m *TunnelManager) Open(tunnel Tunnel) {
	id := tunnelID(tunnel)

	m.mapNotifier.Set(id, -1)

	// Build Args
	args := []string{
		"-v",                             // necesario para detectar cuándo el forwarding está listo
		"-N",                             // No remote command — stay alive for port forwarding only.
		"-o", "ExitOnForwardFailure=yes", // Die immediately if forwarding fails.
		"-F", "/dev/null", // Ignore host-level forwarding directives.
		// Keep each tunnel in its own SSH process. This prevents ControlMaster
		// multiplexing from collapsing multiple tunnel lifecycles into one.
		"-o", "ControlMaster=no",
		"-o", "ServerAliveInterval=30", // Detect broken connections.
		"-o", "ServerAliveCountMax=3",
	}
	switch tunnel.Type {
	case TunnelTypeLocal:
		args = append(args, "-L", tunnel.LocalPort+":"+tunnel.RemoteHost+":"+tunnel.RemotePort)
	case TunnelTypeRemote:
		args = append(args, "-R", tunnel.LocalPort+":"+tunnel.RemoteHost+":"+tunnel.RemotePort)
	case TunnelTypeDynamic:
		args = append(args, "-D", tunnel.LocalPort)
	}

	// PrepareCmd ssh command
	h := GetHost(tunnel.Host)
	slog.Info("Opening", "tunnel", tunnel)
	cmd := PrepareCmd(h, args...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("Error getting stderr pipe", "err", err)
		m.mapNotifier.Unset(id)
		return
	}

	if err := cmd.Start(); err != nil {
		slog.Error("Error starting tunnel", "err", err)
		m.mapNotifier.Unset(id)
		return
	}

	// Lee stderr y detecta cuándo el forwarding está activo (event-driven, sin polling)
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			slog.Debug("ssh", "id", id, "stderr", line)
			if isForwardingReady(tunnel.Type, line) {
				slog.Info("tunnel forwarding ready", "id", id, "pid", cmd.Process.Pid)
				cmd.CleanFn()
				m.mapNotifier.Set(id, cmd.Process.Pid)
			}
		}
	}()

	// Espera a que el proceso termine
	go func() {
		if err := cmd.Wait(); err != nil {
			slog.Error("tunnel exited", "id", id, "error", err)
		}
		m.mapNotifier.Unset(id)
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
	id := tunnelID(tunnel)
	slog.Info("closing tunnel", "id", id)
	err := KillPortOwner(tunnel.LocalPort)
	m.mapNotifier.Unset(id)
	return err
}

func (m *TunnelManager) Status(tunnel Tunnel) (TunnelStatus, PortStatus) {
	id := tunnelID(tunnel)
	tunnelPid, exists := m.mapNotifier.Get(id)
	portStatus, _ := GetPortStatus(tunnel.LocalPort)

	tunnelStatus := TunnelStatus{PID: tunnelPid}
	switch {
	case !exists:
		tunnelStatus.State = TunnelClosed
	case tunnelPid < 1:
		tunnelStatus.State = TunnelOpenning
	default:
		tunnelStatus.State = TunnelOpened
	}

	slog.Info("Tunnel status", "tunnelStatus", tunnelStatus, "portStatus", portStatus)
	return tunnelStatus, portStatus
}

/*
**************

	Persistence

**************
*/

// isForwardingReady detecta la línea de stderr que confirma que SSH
// ya está escuchando en el puerto local.
func isForwardingReady(tunnelType TunnelType, line string) bool {
	switch tunnelType {
	case TunnelTypeLocal, TunnelTypeDynamic:
		return strings.Contains(line, "Local forwarding listening on")
	case TunnelTypeRemote:
		return strings.Contains(line, "remote forward success")
	}
	return false
}

// statePath devuelve la ruta del fichero de estado siguiendo XDG Base Dir Spec.
// Por defecto: ~/.local/share/ssx/tunnels.json
func statePath() string {
	return filepath.Join(paths.DataDir(), "tunnels.json")
}

// saveState persiste en disco los túneles confirmados (PID > 0).
// Se llama automáticamente en cada cambio del mapa.
func (m *TunnelManager) saveState() {
	snapshot := m.mapNotifier.Snapshot()
	state := make(map[string]int, len(snapshot))
	for k, v := range snapshot {
		if v > 0 { // solo túneles confirmados, no los que están en "opening"
			state[k] = v
		}
	}
	if err := atomicfile.WriteJSON(statePath(), state); err != nil {
		slog.Error("failed to save tunnel state", "err", err)
	}
}

// loadState carga el estado persistido y restaura solo los túneles cuyo
// proceso sigue vivo y sigue siendo dueño del puerto local.
func (m *TunnelManager) loadState() {
	var state map[string]int
	if err := atomicfile.ReadJSON(statePath(), &state); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Error("failed to load tunnel state", "err", err)
		}
		return
	}

	for id, pid := range state {
		// tunnelID tiene formato: "host|type|localPort|remoteHost|remotePort"
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

		// Solo restaurar si el mismo proceso sigue escuchando en el puerto
		if portStatus.State == PortOpened && portStatus.PID == pid {
			slog.Info("restoring tunnel from persisted state", "id", id, "pid", pid)
			m.mapNotifier.Set(id, pid)
		} else {
			slog.Info("discarding stale tunnel state", "id", id, "saved_pid", pid, "port_pid", portStatus.PID)
		}
	}
}

/*
**************

	Utils

**************
*/
// ID returns a stable, unique key for this tunnel scoped to a host alias.
// Used as map keys in the tunnel manager.
func tunnelID(t Tunnel) string {
	return fmt.Sprintf("%s|%d|%s|%s|%s",
		t.Host, t.Type, t.LocalPort, t.RemoteHost, t.RemotePort)
}
