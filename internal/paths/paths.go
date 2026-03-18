// Package paths centralizes system path resolution:
// Linux→Windows conversion (WSL), ~ expansion, XDG directories, etc.
package paths

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Home returns the current user's home directory.
// Uses os.UserHomeDir() for cross-platform support, with env var fallbacks.
func Home() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return home
}

// ExpandTilde expands the "~/" prefix to the home directory.
// If the path does not start with "~/", it is returned unchanged.
func ExpandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(Home(), path[2:])
	}
	return path
}

// ToWindows converts a path to its Windows equivalent.
// On WSL it uses wslpath; on native Windows the path is already correct.
// On other systems it returns the original path.
func ToWindows(linuxPath string) string {
	expanded := ExpandTilde(linuxPath)
	if runtime.GOOS == "windows" {
		return expanded
	}
	// On WSL (reports as Linux), delegate to wslpath.
	out, err := exec.Command("wslpath", "-w", expanded).Output()
	if err != nil {
		return expanded
	}
	return strings.TrimSpace(string(out))
}

// AppConfigDir returns the ssx configuration directory.
// $XDG_CONFIG_HOME/ssx or ~/.config/ssx
func AppConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ssx")
	}
	return filepath.Join(Home(), ".config", "ssx")
}

// AppConfigFile returns the full path to the ssx configuration file.
func AppConfigFile() string {
	return filepath.Join(AppConfigDir(), "config.yaml")
}

// AppConfigSampleFile returns the full path to the ssx sample configuration file,
// which is updated on each new version of ssx.
func AppConfigSampleFile() string {
	return filepath.Join(AppConfigDir(), "sample.config.yaml")
}

// SSHConfigFile returns the path to the user's SSH configuration file.
func SSHConfigFile() string {
	return filepath.Join(Home(), ".ssh", "config")
}

// SSHDir returns the ~/.ssh directory.
func SSHDir() string {
	return filepath.Join(Home(), ".ssh")
}

// DataDir returns the ssx data directory.
// $XDG_DATA_HOME/ssx or ~/.local/share/ssx
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "ssx")
	}
	return filepath.Join(Home(), ".local", "share", "ssx")
}

// CacheDir returns the ssx cache directory.
// Windows: %LocalAppData%\ssx | Linux/macOS: $XDG_CACHE_HOME/ssx or ~/.cache/ssx
func CacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "ssx")
	}
	return filepath.Join(Home(), ".cache", "ssx")
}
