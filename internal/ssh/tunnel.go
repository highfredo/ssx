package ssh

import (
	"fmt"
	"strings"
)

// TunnelType identifies the kind of port forwarding a tunnel performs.
type TunnelType int

const (
	TunnelTypeLocal   TunnelType = iota // -L: local port → remote host:port
	TunnelTypeRemote                    // -R: remote port → local host:port
	TunnelTypeDynamic                   // -D: SOCKS5 proxy on local port
)

// Tunnel is a port-forwarding definition bound to a host alias.
type Tunnel struct {
	Type        TunnelType
	Host        string // Alias of the host the tunnel goes through.
	LocalPort   string // Bind port on the local machine.
	RemoteHost  string // Destination host (empty for dynamic).
	RemotePort  string // Destination port (empty for dynamic).
	Description string // Comment of the tunnel (if any).
}

// quickTunnelDescription labels tunnels not declared in ~/.ssh/config.
const quickTunnelDescription = "⚡Quick Tunnel"

// ID returns a stable unique key for the tunnel, scoped to its host alias.
func (t Tunnel) ID() string {
	return fmt.Sprintf("%s|%d|%s|%s|%s", t.Host, t.Type, t.LocalPort, t.RemoteHost, t.RemotePort)
}

// forwardArgs returns the ssh flag that sets up this tunnel.
func (t Tunnel) forwardArgs() []string {
	switch t.Type {
	case TunnelTypeLocal:
		return []string{"-L", t.LocalPort + ":" + t.RemoteHost + ":" + t.RemotePort}
	case TunnelTypeRemote:
		return []string{"-R", t.LocalPort + ":" + t.RemoteHost + ":" + t.RemotePort}
	case TunnelTypeDynamic:
		return []string{"-D", t.LocalPort}
	}
	return nil
}

// TunnelState describes whether a tunnel's local port is free, held by ssx
// or held by an unrelated process.
type TunnelState int

const (
	TunnelUnknown TunnelState = iota
	TunnelClosed
	TunnelOpening
	TunnelOpened
	TunnelPortInUse
)

// TunnelStatus is the observed runtime state of a tunnel.
type TunnelStatus struct {
	State TunnelState
	PID   int32  // Process holding the local port, if any.
	Owner string // Name of that process.
	Err   error  // Last error while opening or closing the tunnel.
}

// ParseTunnel parses a quick-tunnel specification and returns a local Tunnel
// for the given host alias if valid. Supported formats:
//   - "8080"                     → 8080:localhost:8080
//   - "8080:8081"                → 8080:localhost:8081
//   - "8085:internal.net"        → 8085:internal.net:8085
//   - "internal.net:8085"        → 8085:internal.net:8085
//   - "8080:internal.net:8085"   → 8080:internal.net:8085
func ParseTunnel(host, s string) *Tunnel {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	localPort, remoteHost, remotePort, ok := parseQuickTunnelSpec(s)
	if !ok {
		return nil
	}
	return &Tunnel{
		Type:        TunnelTypeLocal,
		Host:        host,
		LocalPort:   localPort,
		RemoteHost:  remoteHost,
		RemotePort:  remotePort,
		Description: quickTunnelDescription,
	}
}

func parseQuickTunnelSpec(s string) (localPort, remoteHost, remotePort string, ok bool) {
	first, rest, hasColon := strings.Cut(s, ":")

	// "8080"
	if !hasColon {
		if !isValidPort(s) {
			return "", "", "", false
		}
		return s, "localhost", s, true
	}

	// "host:port"
	if !isValidPort(first) {
		rh, rp, err := splitHostPort(s)
		if err != nil || !isValidHost(rh) || !isValidPort(rp) {
			return "", "", "", false
		}
		return rp, rh, rp, true
	}

	// "8080:8081"
	if isValidPort(rest) {
		return first, "localhost", rest, true
	}

	// "8080:host:port"
	if rh, rp, err := splitHostPort(rest); err == nil {
		if !isValidHost(rh) || !isValidPort(rp) {
			return "", "", "", false
		}
		return first, rh, rp, true
	}

	// "8080:host" (host may be a bracketed IPv6 literal)
	rh := strings.TrimSuffix(strings.TrimPrefix(rest, "["), "]")
	if !isValidHost(rh) {
		return "", "", "", false
	}
	return first, rh, first, true
}

// tunnelFromCmdline rebuilds a Tunnel from the command line of a running ssh
// process by looking for a -L spec bound to localPort. When none matches, a
// bare quick tunnel for localPort is returned.
func tunnelFromCmdline(hostName, localPort, cmdline string) Tunnel {
	fields := strings.Fields(cmdline)
	for i, arg := range fields {
		var spec string
		switch after, ok := strings.CutPrefix(arg, "-L"); {
		case ok && after != "":
			spec = after
		case ok && i+1 < len(fields):
			spec = fields[i+1]
		default:
			continue
		}
		if tun := ParseTunnel(hostName, spec); tun != nil && tun.LocalPort == localPort {
			return *tun
		}
	}
	return Tunnel{
		Type:        TunnelTypeLocal,
		Host:        hostName,
		LocalPort:   localPort,
		Description: quickTunnelDescription,
	}
}
