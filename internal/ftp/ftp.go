// Package ftp launches an external SFTP client (WinSCP or FileZilla) for a host.
package ftp

import (
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os/exec"
	"strings"

	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/ssh"
)

// ErrNoFTPClient is returned when no SFTP client is installed.
var ErrNoFTPClient = errors.New("no FTP client installed")

type clientSpec struct {
	name   string
	paths  []string
	launch func(*ssh.HostConfig, string) error
}

var clients = []clientSpec{
	{"winscp", winscpPaths, launchWinSCP},
	{"filezilla", filezillaPaths, launchFilezilla},
}

// Launch opens an SFTP connection to h using the client selected by cfg.
// If no explicit client is configured, it tries WinSCP first and FileZilla next.
func Launch(h *ssh.HostConfig, cfg appconfig.FTPClientConfig) error {
	want := strings.ToLower(cfg.Client)

	for _, c := range clients {
		if want != "" && want != c.name {
			continue
		}
		if want == "" && cfg.Path != "" && !strings.Contains(strings.ToLower(cfg.Path), c.name) {
			continue
		}
		if p := resolveBinary(cfg.Path, c.paths); p != "" {
			return c.launch(h, p)
		}
	}
	return ErrNoFTPClient
}

func resolveBinary(customPath string, candidates []string) string {
	if customPath != "" {
		if p, err := exec.LookPath(customPath); err == nil {
			return p
		}
		slog.Warn("custom FTP client path not found", "path", customPath)
		return ""
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}

	return ""
}

// resolvePassword returns the host password, logging and ignoring failures
// so the client still opens and prompts the user.
func resolvePassword(h *ssh.HostConfig) string {
	pw, err := h.ResolvePassword()
	if err != nil {
		slog.Warn("cannot resolve password for SFTP", "host", h.Name, "err", err)
	}
	return pw
}

// buildURL constructs a properly escaped sftp://[user[:pass]@]host[:port]/ URL.
func buildURL(user, host, port, password string) string {
	hostPort := host
	if port != "" {
		hostPort = net.JoinHostPort(host, port)
	}
	u := &url.URL{
		Scheme: "sftp",
		Host:   hostPort,
		Path:   "/",
	}
	if user != "" {
		if password != "" {
			u.User = url.UserPassword(user, password)
		} else {
			u.User = url.User(user)
		}
	}
	return u.String()
}
