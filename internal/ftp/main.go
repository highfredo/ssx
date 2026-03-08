package ftp

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/paths"
	"github.com/highfredo/ssx/internal/ppk"
	"github.com/highfredo/ssx/internal/ssh"
)

// ErrNoFTPClient se devuelve cuando no hay ningún cliente SFTP instalado.
var ErrNoFTPClient = errors.New("no FTP client installed")

// Launch abre la conexión SFTP con el cliente configurado en AppConfig.
// Si no hay configuración explícita, prueba WinSCP primero y FileZilla después.
// Devuelve ErrNoFTPClient si no hay ningún cliente disponible.
func Launch(h *ssh.HostConfig) error {
	cfg := appconfig.Get().FTP

	switch strings.ToLower(cfg.Client) {
	case "winscp":
		p := resolveBinary(cfg.Path, winscpPaths)
		if p == "" {
			return fmt.Errorf("%w: WinSCP not found", ErrNoFTPClient)
		}
		return launchWinSCP(h, p)

	case "filezilla":
		p := resolveBinary(cfg.Path, filezillaPaths)
		if p == "" {
			return fmt.Errorf("%w: FileZilla not found", ErrNoFTPClient)
		}
		return launchFilezilla(h, p)

	default: // auto-detect
		if cfg.Path != "" {
			// Ruta personalizada sin cliente explícito: detectar por nombre
			return launchByPath(h, cfg.Path)
		}
		if p := findBinary(winscpPaths); p != "" {
			return launchWinSCP(h, p)
		}
		if p := findBinary(filezillaPaths); p != "" {
			return launchFilezilla(h, p)
		}
		return ErrNoFTPClient
	}
}

// resolveBinary devuelve customPath si está definido y existe, o busca en candidates.
func resolveBinary(customPath string, candidates []string) string {
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			return customPath
		}
		slog.Warn("ruta de cliente FTP no encontrada", "path", customPath)
		return ""
	}
	return findBinary(candidates)
}

// launchByPath lanza el cliente adecuado deduciendo el tipo por el nombre del ejecutable.
func launchByPath(h *ssh.HostConfig, exePath string) error {
	lower := strings.ToLower(exePath)
	switch {
	case strings.Contains(lower, "winscp"):
		return launchWinSCP(h, exePath)
	case strings.Contains(lower, "filezilla"):
		return launchFilezilla(h, exePath)
	default:
		return fmt.Errorf("%w: no se puede determinar el tipo de cliente para %q", ErrNoFTPClient, exePath)
	}
}

// findBinary devuelve la primera ruta de la lista que existe en disco, o "".
func findBinary(candidates []string) string {
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// buildURL construye la URL sftp://[user[:pass]@]host:port/.
func buildURL(user, host, port, password string) string {
	if user == "" {
		return fmt.Sprintf("sftp://%s:%s/", host, port)
	}
	if password != "" {
		return fmt.Sprintf("sftp://%s:%s@%s:%s/", user, password, host, port)
	}
	return fmt.Sprintf("sftp://%s@%s:%s/", user, host, port)
}

// resolveKeyPath devuelve la ruta de la clave privada efectiva del host.
func resolveKeyPath(h *ssh.HostConfig) string {
	if len(h.IdentityFiles) > 0 {
		return paths.ExpandTilde(h.IdentityFiles[0])
	}
	if k, err := ssh.FindDefaultKey(); err == nil {
		return k
	}
	return ""
}

// resolvePassword devuelve la contraseña efectiva del host:
// usa Password si está definida, o ejecuta PasswordCommand para obtenerla.
func resolvePassword(h *ssh.HostConfig) string {
	if h.Password != "" {
		return h.Password
	}
	if h.PasswordCommand == "" {
		return ""
	}
	out, err := exec.Command("sh", "-c", h.PasswordCommand).Output()
	if err != nil {
		slog.Warn("passwordCommand falló", "cmd", h.PasswordCommand, "err", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ensurePpk convierte una clave OpenSSH a formato PPK v2.
// Si ya es .ppk o la conversión falla, devuelve la ruta original.
func ensurePpk(keyPath string) string {
	if strings.HasSuffix(keyPath, ".ppk") {
		return keyPath // ya es formato PuTTY
	}

	data, err := os.ReadFile(keyPath)
	if err != nil || !strings.Contains(string(data), "BEGIN OPENSSH PRIVATE KEY") {
		return keyPath // formato antiguo, WinSCP lo acepta tal cual
	}

	ppkPath := keyPath + ".ppk"

	// Regenerar solo si el .ppk no existe o la clave original es más nueva
	keyStat, _ := os.Stat(keyPath)
	ppkStat, ppkErr := os.Stat(ppkPath)
	if ppkErr == nil && !keyStat.ModTime().After(ppkStat.ModTime()) {
		return ppkPath // .ppk cacheado y vigente
	}

	ppkData, err := ppk.FromOpenSSH(data, filepath.Base(keyPath))
	if err != nil {
		slog.Warn("conversión PPK falló, usando clave original", "key", keyPath, "err", err)
		return keyPath
	}

	if err := os.WriteFile(ppkPath, ppkData, 0600); err != nil {
		slog.Warn("no se pudo escribir el archivo PPK", "ppk", ppkPath, "err", err)
		return keyPath
	}

	slog.Info("clave convertida a PPK v2", "ppk", ppkPath)
	return ppkPath
}
