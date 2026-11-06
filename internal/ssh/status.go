package ssh

import (
	"log/slog"
	"maps"
	"slices"
)

// TunnelReport pairs a tunnel with its observed status.
type TunnelReport struct {
	Tunnel Tunnel
	Status TunnelStatus
}

// GetTunnelStatus inspects the local listening TCP ports and reports whether
// the tunnel's local port is free, held by an ssh process for host (i.e.
// opened by ssx) or held by another process. A nil host accepts any ssh
// process as the tunnel owner.
func GetTunnelStatus(host *HostConfig, tunnel Tunnel) (TunnelStatus, error) {
	port, err := parsePort(tunnel.LocalPort)
	if err != nil {
		return TunnelStatus{State: TunnelClosed}, err
	}
	snap, err := snapshotPorts()
	if err != nil {
		return TunnelStatus{State: TunnelClosed}, err
	}
	return snap.tunnelStatus(host, port), nil
}

// HostTunnels returns the tunnels configured for host plus any ssh tunnel
// currently listening on a local port for host that is not declared in
// ~/.ssh/config, each paired with its observed status.
func HostTunnels(host *HostConfig) []TunnelReport {
	if host == nil {
		return nil
	}
	snap, err := snapshotPorts()
	if err != nil {
		slog.Warn("HostTunnels: cannot list tcp connections", "err", err)
		return unknownReports(host.Tunnels)
	}

	tunnels := slices.Clone(host.Tunnels)
	known := make(map[string]bool, len(tunnels))
	for _, t := range tunnels {
		known[t.LocalPort] = true
	}
	for _, t := range snap.openTunnelsForHost(host) {
		if !known[t.LocalPort] {
			tunnels = append(tunnels, t)
			known[t.LocalPort] = true
		}
	}
	return snap.reports(host, tunnels)
}

// OpenTunnels returns every tunnel (configured or discovered) whose local
// port is currently held by an ssh process for one of hosts.
func OpenTunnels(hosts Hosts) []TunnelReport {
	snap, err := snapshotPorts()
	if err != nil {
		slog.Warn("OpenTunnels: cannot list tcp connections", "err", err)
		return nil
	}
	var reports []TunnelReport
	for _, h := range hosts {
		reports = append(reports, snap.reports(h, snap.openTunnelsForHost(h))...)
	}
	return reports
}

func unknownReports(tunnels []Tunnel) []TunnelReport {
	reports := make([]TunnelReport, len(tunnels))
	for i, t := range tunnels {
		reports[i] = TunnelReport{Tunnel: t, Status: TunnelStatus{State: TunnelUnknown}}
	}
	return reports
}

// reports evaluates the status of each tunnel against the snapshot.
func (s *portSnapshot) reports(host *HostConfig, tunnels []Tunnel) []TunnelReport {
	reports := make([]TunnelReport, len(tunnels))
	for i, t := range tunnels {
		reports[i] = TunnelReport{Tunnel: t, Status: s.tunnelStatusFor(host, t)}
	}
	return reports
}

func (s *portSnapshot) tunnelStatusFor(host *HostConfig, tunnel Tunnel) TunnelStatus {
	port, err := parsePort(tunnel.LocalPort)
	if err != nil {
		return TunnelStatus{State: TunnelClosed}
	}
	return s.tunnelStatus(host, port)
}

// tunnelStatus classifies the first visible owner of port.
func (s *portSnapshot) tunnelStatus(host *HostConfig, port uint32) TunnelStatus {
	owners := s.ownersOf(port)
	if len(owners) == 0 {
		return TunnelStatus{State: TunnelClosed}
	}
	o := owners[0]
	state := TunnelPortInUse
	if o.isSSH() && cmdlineMatchesHost(o.Cmdline, host) {
		state = TunnelOpened
	}
	return TunnelStatus{State: state, PID: o.PID, Owner: o.Name}
}

// openTunnelsForHost returns the configured tunnels of host that are currently
// held by an ssh process targeting host, followed by tunnels reconstructed
// from any other ssh process targeting host (sorted by port).
func (s *portSnapshot) openTunnelsForHost(host *HostConfig) []Tunnel {
	cmdlines := s.sshCmdlines()
	var result []Tunnel
	matched := make(map[string]bool)

	for _, t := range host.Tunnels {
		if cmdline, ok := cmdlines[t.LocalPort]; ok && cmdlineMatchesHost(cmdline, host) {
			result = append(result, t)
			matched[t.LocalPort] = true
		}
	}
	for _, port := range slices.Sorted(maps.Keys(cmdlines)) {
		if matched[port] || !cmdlineMatchesHost(cmdlines[port], host) {
			continue
		}
		result = append(result, tunnelFromCmdline(host.Name, port, cmdlines[port]))
	}
	return result
}
