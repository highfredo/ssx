package ftp

import (
	"log/slog"

	"github.com/highfredo/ssx/internal/paths"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/system"
)

var winscpPaths = []string{
	"/mnt/c/Program Files (x86)/WinSCP/winscp.exe",
	"/mnt/c/Program Files/WinSCP/winscp.exe",
}

// launchWinSCP abre WinSCP con soporte completo de clave privada y contraseña.
// Solo funciona en entornos Windows/WSL; la clave se convierte a PPK v2
// y las rutas se convierten al formato Windows con wslpath.
func launchWinSCP(h *ssh.HostConfig, exe string) error {
	args := []string{buildURL(h.User, h.Hostname, h.Port, "")}

	if keyPath := resolveKeyPath(h); keyPath != "" {
		args = append(args, "/privatekey="+paths.ToWindows(ensurePpk(keyPath)))
	}
	if password := resolvePassword(h); password != "" {
		args = append(args, "/password="+password)
	}

	slog.Info("launching WinSCP", "host", h.Hostname)
	_, err := system.Run(exe, args...)
	return err
}
