package ftp

import (
	"log/slog"

	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/system"
)

// launchFilezilla opens FileZilla directly with the SFTP URL.
func launchFilezilla(h *ssh.HostConfig, exe string) error {
	slog.Info("launching FileZilla", "host", h.Hostname)
	sftpURL := buildURL(h.User, h.Hostname, h.Port, resolvePassword(h))
	return system.Start(exe, sftpURL)
}
