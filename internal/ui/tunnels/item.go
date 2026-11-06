package tunnels

import (
	"fmt"
	"strings"

	"github.com/highfredo/ssx/internal/ssh"
)

type item struct {
	tunnel ssh.Tunnel
	status ssh.TunnelStatus
	quick  bool // virtual item built from the search query, not yet opened
}

// Title renders the forwarding spec as plain text so the delegate can
// highlight filter matches on it. The status badge is drawn by the delegate.
func (i item) Title() string {
	t := i.tunnel
	switch t.Type {
	case ssh.TunnelTypeLocal:
		if t.RemoteHost == "" && t.RemotePort == "" {
			return t.LocalPort
		}
		return fmt.Sprintf("%s → %s:%s", t.LocalPort, t.RemoteHost, t.RemotePort)
	case ssh.TunnelTypeRemote:
		return fmt.Sprintf("%s ← %s:%s", t.LocalPort, t.RemoteHost, t.RemotePort)
	case ssh.TunnelTypeDynamic:
		return fmt.Sprintf("%s ⇄ *:*", t.LocalPort)
	default:
		return t.LocalPort
	}
}

// Description shows the user description or, failing that, the forward kind.
// Runtime details (port owner, last error) are appended by the delegate so
// they can be coloured independently.
func (i item) Description() string {
	if i.quick {
		return "Quick tunnel  ·  enter to open"
	}
	if i.tunnel.Description != "" {
		return i.tunnel.Description
	}
	return typeName(i.tunnel.Type)
}

// FilterValue starts with Title so the fuzzy-match indexes line up with the
// rendered title and the highlight underlines the right runes.
func (i item) FilterValue() string {
	t := i.tunnel
	parts := []string{
		i.Title(),
		fmt.Sprintf("%s:%s:%s", t.LocalPort, t.RemoteHost, t.RemotePort),
		fmt.Sprintf("%s:%s", t.LocalPort, t.RemotePort),
		fmt.Sprintf("%s:%s", t.LocalPort, t.RemoteHost),
		fmt.Sprintf("%s:%s", t.RemoteHost, t.RemotePort),
		t.LocalPort,
		t.RemoteHost,
		t.RemotePort,
		t.Description,
		typeName(t.Type),
		t.Host,
	}
	return strings.Join(parts, " ")
}

func typeName(t ssh.TunnelType) string {
	switch t {
	case ssh.TunnelTypeLocal:
		return "Local forward"
	case ssh.TunnelTypeRemote:
		return "Remote forward"
	case ssh.TunnelTypeDynamic:
		return "Dynamic SOCKS proxy"
	}
	return ""
}
