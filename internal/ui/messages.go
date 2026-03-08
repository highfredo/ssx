package ui

import "github.com/highfredo/ssx/internal/sshconfig"

// SSHExitedMsg is sent when an interactive SSH session (opened with 'c') ends.
type SSHExitedMsg struct {
	Err error
}

type requestConnectMsg struct {
	host *sshconfig.Host
}

// openTunnelViewMsg navigates to the tunnel management screen for a host.
type openTunnelViewMsg struct {
	host *sshconfig.Host
}

// backMsg navigates back to the host list from the tunnel view.
type backMsg struct{}

// openTunnelResultMsg reports the result of an Open tunnel request.
type openTunnelResultMsg struct {
	TunnelID string
	Err      error
}

type requestOpenTunnelMsg struct {
	hostName string
	tunnel   sshconfig.Tunnel
}

// closeTunnelResultMsg reports the result of a Close tunnel request.
type closeTunnelResultMsg struct {
	TunnelID string
	Err      error
}

// refreshTunnelViewMsg triggers a tunnel-view re-render after an async event.
type refreshTunnelViewMsg struct{}
