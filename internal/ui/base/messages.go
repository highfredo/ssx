package base

import "github.com/highfredo/ssx/internal/ssh"

type OpenTunnelForHostPageMsg struct {
	HostConfig *ssh.HostConfig
}

type OpenTunnelsOpenPageMsg struct{}

type OpenHostPageMsg struct{}

type OpenInfoPageMsg struct {
	HostConfig *ssh.HostConfig
}

type OpenModalMsg struct {
	Dialog Component
}

type CloseModalMsg struct{}

// AppUpdatedMsg is sent when ssx has successfully updated itself in the background.
// NewVersion holds the version string that was installed.
type AppUpdatedMsg struct {
	NewVersion string
}

