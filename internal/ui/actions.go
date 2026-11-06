package ui

import (
	"errors"
	"fmt"
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/ftp"
	"github.com/highfredo/ssx/internal/paths"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/system"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/modal"
)

// copySSHKeyMsg is emitted once the user confirms the ssh-copy-id dialog.
type copySSHKeyMsg struct {
	host *ssh.HostConfig
}

// confirmCopySSHKey asks for confirmation before installing the public key.
func confirmCopySSHKey(h *ssh.HostConfig) tea.Cmd {
	slog.Info("ssh-copy-id confirm", "name", h.Name)
	return modal.OpenConfirm(
		"Copy SSH key",
		fmt.Sprintf("Copy your public key to %s (%s)?", h.Name, h.Hostname),
		func(confirmed bool) tea.Cmd {
			if !confirmed {
				return nil
			}
			return base.Emit(copySSHKeyMsg{host: h})
		},
	)
}

// copySSHKey installs the public key on the host in the background. When
// ssh rejects the credentials it asks for a password and retries.
func copySSHKey(h *ssh.HostConfig) tea.Cmd {
	return tea.Batch(
		modal.OpenSpinner("Copying SSH key…"),
		func() tea.Msg {
			err := ssh.RunCopyID(h)
			switch {
			case errors.Is(err, ssh.ErrAuthFailed):
				slog.Warn("ssh copy-id: authentication failed", "host", h.Hostname)
				return askPassword(h, copySSHKey)()
			case err != nil:
				slog.Error("ssh copy-id failed", "host", h.Hostname, "err", err)
				return modal.OpenAlert("Cannot copy SSH key", err.Error(), true)()
			}
			slog.Info("ssh copy-id succeeded", "host", h.Hostname)
			return modal.OpenAlert(
				"Key copied successfully",
				fmt.Sprintf("The public key was copied to %s.", h.Hostname),
				false,
			)()
		},
	)
}

// askPassword opens the password modal for h and, on submit, runs retry with
// a copy of the host that authenticates with the entered password.
func askPassword(h *ssh.HostConfig, retry func(*ssh.HostConfig) tea.Cmd) tea.Cmd {
	return modal.OpenPassword(
		"Authentication failed",
		fmt.Sprintf("Enter the password for %s.", h.Target()),
		func(pw string) tea.Cmd { return retry(h.WithPassword(pw)) },
	)
}

// openFTP launches the configured SFTP client, showing a spinner meanwhile.
func openFTP(h *ssh.HostConfig, cfg appconfig.FTPClientConfig) tea.Cmd {
	slog.Info("opening SFTP client", "name", h.Name)
	return tea.Batch(
		modal.OpenSpinner("Opening SFTP client…"),
		func() tea.Msg {
			err := ftp.Launch(h, cfg)
			switch {
			case errors.Is(err, ftp.ErrNoFTPClient):
				return modal.OpenAlert("No SFTP client found", "Please install WinSCP or FileZilla", true)()
			case err != nil:
				return modal.OpenAlert("Error opening SFTP client", err.Error(), true)()
			}
			return base.CloseModalMsg{}
		},
	)
}

// editSSHConfig opens the host's config file at its Host line in $EDITOR.
func editSSHConfig(h *ssh.HostConfig) tea.Cmd {
	cfgFile := h.ConfigFile
	if cfgFile == "" {
		cfgFile = paths.SSHConfigFile()
	}
	slog.Info("editing ssh config", "file", cfgFile, "line", h.ConfigLine)
	return func() tea.Msg {
		if err := system.EditFile(cfgFile, h.ConfigLine); err != nil {
			return modal.OpenAlert("Cannot edit SSH config", err.Error(), true)()
		}
		return nil
	}
}
