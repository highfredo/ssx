package ftp

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/highfredo/ssx/internal/paths"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/system"
)

// launchWinSCP opens WinSCP with private key and password support.
// On Windows/WSL, OpenSSH keys are converted to PPK v2 and paths are converted with wslpath.
func launchWinSCP(h *ssh.HostConfig, exe string) error {
	args := []string{buildURL(h.User, h.Hostname, h.Port, resolvePassword(h))}

	if keyPath := resolveKeyPath(h); keyPath != "" {
		args = append(args, "/privatekey="+paths.ToWindows(ensurePpk(keyPath)))
	}

	slog.Info("launching WinSCP", "host", h.Hostname)
	return system.Start(exe, args...)
}

func ensurePpk(keyPath string) string {
	if strings.HasSuffix(strings.ToLower(keyPath), ".ppk") {
		return keyPath
	}

	data, err := os.ReadFile(keyPath)
	if err != nil || !strings.Contains(string(data), "BEGIN OPENSSH PRIVATE KEY") {
		return keyPath
	}

	ppkPath := keyPath + ".ppk"

	keyStat, err := os.Stat(keyPath)
	if err == nil {
		if ppkStat, ppkErr := os.Stat(ppkPath); ppkErr == nil && !keyStat.ModTime().After(ppkStat.ModTime()) {
			return ppkPath
		}
	}

	ppkData, err := ssh.OpenSSHKeyToPuttyKey(data, filepath.Base(keyPath))
	if err != nil {
		slog.Warn("PPK conversion failed, using original key", "key", keyPath, "err", err)
		return keyPath
	}

	if err := os.WriteFile(ppkPath, ppkData, 0o600); err != nil {
		slog.Warn("failed to write PPK file", "ppk", ppkPath, "err", err)
		return keyPath
	}

	slog.Info("converted key to PPK v2", "ppk", ppkPath)
	return ppkPath
}

func resolveKeyPath(h *ssh.HostConfig) string {
	for _, id := range h.IdentityFiles {
		p := paths.ExpandTilde(id)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if k, err := ssh.FindDefaultKey(); err == nil {
		return k
	}
	return ""
}
