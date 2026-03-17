// Package paths centraliza la resolución de rutas del sistema:
// conversión Linux→Windows (WSL), expansión de ~, directorios XDG, etc.
package paths

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Home devuelve el directorio home del usuario actual.
func Home() string {
	return os.Getenv("HOME")
}

// ExpandTilde expande el prefijo "~/" al directorio home.
// Si la ruta no empieza por "~/", la devuelve sin cambios.
func ExpandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(Home(), path[2:])
	}
	return path
}

// ToWindows convierte una ruta Linux a su equivalente Windows usando wslpath.
// Si la conversión falla devuelve la ruta original.
func ToWindows(linuxPath string) string {
	out, err := exec.Command("wslpath", "-w", ExpandTilde(linuxPath)).Output()
	if err != nil {
		return linuxPath
	}
	return strings.TrimSpace(string(out))
}

// AppConfigDir devuelve el directorio de configuración de ssx,
// respetando XDG_CONFIG_HOME si está definido.
func AppConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ssx")
	}
	return filepath.Join(Home(), ".config", "ssx")
}

// AppConfigFile devuelve la ruta completa al fichero de configuración de ssx.
func AppConfigFile() string {
	return filepath.Join(AppConfigDir(), "config.yaml")
}

// AppConfigSampleFile devuelve la ruta completa al fichero de configuración de ejemplo,
// que se actualiza en cada nueva versión de ssx.
func AppConfigSampleFile() string {
	return filepath.Join(AppConfigDir(), "sample.config.yaml")
}

// SSHConfigFile devuelve la ruta al fichero de configuración SSH del usuario.
func SSHConfigFile() string {
	return filepath.Join(Home(), ".ssh", "config")
}

// SSHDir devuelve el directorio ~/.ssh.
func SSHDir() string {
	return filepath.Join(Home(), ".ssh")
}

// DataDir devuelve el directorio de datos de ssx (~/.local/share/ssx),
// respetando XDG_DATA_HOME si está definido.
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "ssx")
	}
	return filepath.Join(Home(), ".local", "share", "ssx")
}

func CacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "ssx")
	}
	return filepath.Join(Home(), ".cache", "ssx")
}
