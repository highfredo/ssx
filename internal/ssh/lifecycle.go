package ssh

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// tunnelBaseArgs are the ssh flags shared by every tunnel process.
var tunnelBaseArgs = []string{
	"-v",              // Required to detect when forwarding is ready.
	"-N",              // No remote command: stay alive for port forwarding only.
	"-F", "/dev/null", // Ignore ~/.ssh/config so its LocalForward entries aren't opened too.
	"-o", "ControlMaster=no", // Isolate the tunnel from any user ControlMaster.
	"-o", "ControlPath=none", // Ensure a standalone ssh process per tunnel.
	"-o", "ExitOnForwardFailure=yes", // Die immediately if forwarding fails.
	"-o", "ServerAliveInterval=30",
	"-o", "ServerAliveCountMax=3",
}

// OpenTunnel spawns a background ssh process for tunnel through host and
// blocks until port forwarding is established or the process exits.
func OpenTunnel(host *HostConfig, tunnel Tunnel) (TunnelStatus, error) {
	if host == nil {
		return TunnelStatus{State: TunnelClosed}, fmt.Errorf("host %q not found", tunnel.Host)
	}

	args := slices.Concat(tunnelBaseArgs, hostArgs(host))
	if !host.HasPassword() {
		args = append(args, "-o", "BatchMode=yes")
	}
	args = append(args, tunnel.forwardArgs()...)

	slog.Info("opening tunnel", "id", tunnel.ID())
	cmd := PrepareCmd(host, args...)

	ready := make(chan struct{}, 1)
	stderr := newLineWriter(forwardingWatcher(tunnel, ready))
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return failedStatus(err), err
	}

	// cmd.Wait also drains stderr, so the full output is available afterwards.
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	select {
	case <-ready:
		slog.Info("tunnel forwarding ready", "id", tunnel.ID(), "pid", cmd.Process.Pid)
		return GetTunnelStatus(host, tunnel)
	case err := <-exited:
		err = classifyExit(err, stderr.String())
		if err != nil {
			slog.Error("tunnel exited before ready", "id", tunnel.ID(), "err", err)
		}
		st, _ := GetTunnelStatus(host, tunnel)
		st.Err = err
		return st, err
	}
}

func failedStatus(err error) TunnelStatus {
	return TunnelStatus{State: TunnelClosed, Err: err}
}

// forwardingWatcher returns a stderr line handler that signals ready once
// ssh's verbose output shows the forwarding is established.
func forwardingWatcher(tunnel Tunnel, ready chan<- struct{}) func(string) {
	signaled := false
	return func(line string) {
		slog.Debug("ssh", "id", tunnel.ID(), "stderr", line)
		if !signaled && isForwardingReady(tunnel.Type, line) {
			signaled = true
			ready <- struct{}{}
		}
	}
}

func isForwardingReady(tunnelType TunnelType, line string) bool {
	switch tunnelType {
	case TunnelTypeLocal, TunnelTypeDynamic:
		return strings.Contains(line, "Local forwarding listening on")
	case TunnelTypeRemote:
		return strings.Contains(line, "remote forward success")
	}
	return false
}

// hostArgs translates the host configuration into explicit ssh flags, needed
// because tunnels run with "-F /dev/null".
func hostArgs(host *HostConfig) []string {
	var args []string
	if host.Hostname != "" {
		args = append(args, "-o", "HostName="+host.Hostname)
	}
	if host.User != "" {
		args = append(args, "-l", host.User)
	}
	if host.Port != "" {
		args = append(args, "-p", host.Port)
	}
	for _, id := range host.IdentityFiles {
		args = append(args, "-i", id)
	}
	if host.ProxyCommand != "" {
		args = append(args, "-o", "ProxyCommand="+host.ProxyCommand)
	}
	if host.ProxyJump != "" {
		args = append(args, "-J", host.ProxyJump)
	}
	return args
}

// CloseTunnel terminates the process listening on the tunnel's local port.
func CloseTunnel(tunnel Tunnel) error {
	slog.Info("closing tunnel", "id", tunnel.ID())
	return KillPortOwner(tunnel.LocalPort)
}

// KillPortOwner terminates every visible process listening on port and waits
// briefly for the port to be released.
func KillPortOwner(port string) error {
	portNum, err := parsePort(port)
	if err != nil {
		return err
	}
	snap, err := snapshotPorts()
	if err != nil {
		return err
	}

	killed := make(map[int32]bool)
	invisible := false
	for _, o := range snap.ownersOf(portNum) {
		if o.PID <= 0 {
			invisible = true
			continue
		}
		if killed[o.PID] {
			continue
		}
		if err := killProcess(o.PID); err != nil {
			return err
		}
		slog.Info("killed port owner", "port", port, "pid", o.PID, "command", o.Name)
		killed[o.PID] = true
	}

	if len(killed) == 0 && invisible {
		return fmt.Errorf("permission denied: port %s is owned by another user/root", port)
	}
	if len(killed) > 0 {
		waitForPortRelease(portNum)
	}
	return nil
}

func killProcess(pid int32) error {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := proc.Kill(); err != nil {
		return fmt.Errorf("kill process %d: %w", pid, err)
	}
	return nil
}

// waitForPortRelease polls until port is free or a short deadline expires.
func waitForPortRelease(port uint32) {
	const (
		timeout  = 500 * time.Millisecond
		interval = 25 * time.Millisecond
	)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if snap, err := snapshotPorts(); err == nil && snap.isFree(port) {
			return
		}
		time.Sleep(interval)
	}
}
