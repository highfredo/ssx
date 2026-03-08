package tunnels

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/styles"
)

type item struct {
	tunnel       ssh.Tunnel
	tunnelStatus ssh.TunnelStatus
	portStatus   ssh.PortStatus
}

func (i item) Title() string {
	switch i.tunnel.Type {
	case ssh.TunnelTypeLocal:
		return fmt.Sprintf("[L] :%s → %s:%s", i.tunnel.LocalPort, i.tunnel.RemoteHost, i.tunnel.RemotePort)
	case ssh.TunnelTypeRemote:
		return fmt.Sprintf("[R] :%s ← %s:%s", i.tunnel.LocalPort, i.tunnel.RemoteHost, i.tunnel.RemotePort)
	case ssh.TunnelTypeDynamic:
		return fmt.Sprintf("[D] :%s", i.tunnel.LocalPort)
	default:
		return i.tunnel.LocalPort
	}
}

func (i item) Description() string {
	badge := statusBadge(i.tunnelStatus, i.portStatus)
	//if i.tunnel.Description != "" {
	//	return badge + "  " + i.tunnel.Description
	//}

	txt := badge + "  " + i.portStatus.Owner

	if i.tunnelStatus.Err != nil {
		txt += "  " + i.tunnelStatus.Err.Error()
	}

	return txt
}

func (i item) FilterValue() string {
	parts := []string{i.tunnel.LocalPort, i.tunnel.RemoteHost, i.tunnel.RemotePort, i.tunnel.Description}
	return strings.Join(parts, " ")
}

func statusBadge(tunnelStatus ssh.TunnelStatus, portStatus ssh.PortStatus) string {
	if tunnelStatus.State == ssh.TunnelClosed && portStatus.State == ssh.PortOpened {
		return lipgloss.NewStyle().Foreground(styles.ColorDanger).Bold(true).Render("⚠ in use")
	} else if tunnelStatus.State == ssh.TunnelOpenning {
		return lipgloss.NewStyle().Foreground(styles.ColorHighlight).Bold(true).Render("● opening")
	} else if tunnelStatus.State == ssh.TunnelOpened {
		return styles.StateOpenStyle.Render("● open")
	} else if tunnelStatus.State == ssh.TunnelClosed {
		return styles.StateClosedStyle.Render("○ closed")
	}
	return ""
}
