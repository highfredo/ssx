package base

import "github.com/highfredo/ssx/internal/ssh"

// OpenTunnelForHostPageMsg navigates to the tunnel page of a host.
type OpenTunnelForHostPageMsg struct {
	HostConfig *ssh.HostConfig
}

// OpenTunnelsOpenPageMsg navigates to the page listing every open tunnel.
type OpenTunnelsOpenPageMsg struct{}

// OpenHostPageMsg navigates back to the root host list.
type OpenHostPageMsg struct{}

// OpenInfoPageMsg navigates to the resolved ssh config view of a host.
type OpenInfoPageMsg struct {
	HostConfig *ssh.HostConfig
}

// CopySSHKeyMsg requests installing the local public key on a host.
type CopySSHKeyMsg struct {
	HostConfig *ssh.HostConfig
}

// OpenFTPMsg requests launching the SFTP client against a host.
type OpenFTPMsg struct {
	HostConfig *ssh.HostConfig
}

// EditSSHConfigMsg requests opening the host's ssh config block in an editor.
type EditSSHConfigMsg struct {
	HostConfig *ssh.HostConfig
}

// OpenModalMsg shows Dialog on top of the active page until CloseModalMsg.
type OpenModalMsg struct {
	Dialog Component
}

// CloseModalMsg dismisses the current modal, if any.
type CloseModalMsg struct{}

// AppUpdatedMsg is sent when ssx has successfully updated itself in the
// background. NewVersion holds the version string that was installed.
type AppUpdatedMsg struct {
	NewVersion string
}
