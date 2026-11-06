// Package appconfig manages the ssx application configuration,
// read from ~/.config/ssx/config.yaml with XDG support.
package appconfig

import (
	"log/slog"
	"os"

	"github.com/highfredo/ssx/assets"
	"github.com/highfredo/ssx/internal/paths"
	"gopkg.in/yaml.v3"
)

// Config holds the ssx configuration read from ~/.config/ssx/config.yaml.
type Config struct {
	// LogLevel sets the logging level: "debug", "info", "warn" or "error".
	// Empty defaults to "info".
	LogLevel string `yaml:"log_level"`
	// Theme customizes the UI color palette.
	Theme ThemeConfig `yaml:"theme"`
	// Tags defines default colors for each tag.
	Tags map[string]string `yaml:"tags"`
	// FTP configures the preferred SFTP client.
	FTP FTPClientConfig `yaml:"ftp"`
	// Keybindings allows customizing keyboard shortcuts.
	Keybindings KeybindingsConfig `yaml:"keybindings"`
}

// KeybindingsConfig groups all customizable keyboard shortcuts.
// Each field is a space-separated list of keys (e.g. "ctrl+t tab").
// An empty field uses the default value.
type KeybindingsConfig struct {
	// Quit exits the application. Default: "ctrl+c".
	Quit string `yaml:"quit"`
	// Hosts groups the shortcuts of the main hosts screen.
	Hosts HostsKeybindings `yaml:"hosts"`
	// Tunnels groups the shortcuts of the tunnels screen.
	Tunnels TunnelsKeybindings `yaml:"tunnels"`
	// Navigation groups the navigation and scroll shortcuts.
	Navigation NavigationKeybindings `yaml:"navigation"`
}

// HostsKeybindings holds the shortcuts of the hosts screen.
type HostsKeybindings struct {
	Connect      string `yaml:"connect"`       // Default: "enter"
	CopyKey      string `yaml:"copy_key"`      // Default: "ctrl+a"
	Tunnels      string `yaml:"tunnels"`       // Default: "ctrl+t tab"
	OpenTunnels  string `yaml:"open_tunnels"`  // Default: "ctrl+o"
	FTP          string `yaml:"ftp"`           // Default: "ctrl+x"
	Info         string `yaml:"info"`          // Default: "ctrl+g"
	EditConfig   string `yaml:"edit_config"`   // Default: "ctrl+e"
	OpenConfig   string `yaml:"open_config"`   // Default: "ctrl+alt+s"
	QuickConnect string `yaml:"quick_connect"` // Default: "ctrl+enter ctrl+j"
}

// TunnelsKeybindings holds the shortcuts of the tunnels screen.
type TunnelsKeybindings struct {
	Toggle  string `yaml:"toggle"`  // Default: "enter"
	Browser string `yaml:"browser"` // Default: "ctrl+b"
	Back    string `yaml:"back"`    // Default: "esc tab"
}

// NavigationKeybindings holds the navigation shortcuts shared by
// all scrollable lists and views (hosts, tunnels, info).
type NavigationKeybindings struct {
	CursorUp     string `yaml:"cursor_up"`      // Default: "up"
	CursorDown   string `yaml:"cursor_down"`    // Default: "down"
	PrevPage     string `yaml:"prev_page"`      // Default: "pgup"
	NextPage     string `yaml:"next_page"`      // Default: "pgdown"
	GoToStart    string `yaml:"go_to_start"`    // Default: "home"
	GoToEnd      string `yaml:"go_to_end"`      // Default: "end"
	ShowFullHelp string `yaml:"show_full_help"` // Default: "f1 ?"
	HalfPageUp   string `yaml:"half_page_up"`   // Default: "ctrl+u"
	HalfPageDown string `yaml:"half_page_down"` // Default: "ctrl+d"
	Back         string `yaml:"back"`           // Default: "tab esc"
}

// SlogLevel returns the slog.Level corresponding to the LogLevel field.
// Valid values: "debug", "info", "warn", "error" (case-insensitive).
// Any other value is treated as "info".
func (c *Config) SlogLevel() slog.Level {
	switch c.LogLevel {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ThemeConfig customizes the color palette of the ssx interface.
// Each field accepts a hex color (#RRGGBB), a basic CSS color name
// or an ANSI number. Empty fields keep the default color.
type ThemeConfig struct {
	Primary    string `yaml:"primary"`    // accent / title background        (default #7C3AED)
	Secondary  string `yaml:"secondary"`  // selected description, section titles, keys (default #C4B5FD)
	Highlight  string `yaml:"highlight"`  // selected row (dark)               (default #1E1B4B)
	Text       string `yaml:"text"`       // main text on accent / selected    (default #FAFAFA)
	Success    string `yaml:"success"`    // open state / OK                   (default #10B981)
	Warning    string `yaml:"warning"`    // transitional state (opening)      (default #F59E0B)
	Muted      string `yaml:"muted"`      // closed state / help text          (default #6B7280)
	Danger     string `yaml:"danger"`     // errors / port in use              (default #EF4444)
	Border     string `yaml:"border"`     // panel borders / inactive buttons  (default #374151)
	DefaultTag string `yaml:"defaultTag"` // tag chips without explicit colour (default #312E81)
}

// FTPClientConfig configures which FTP/SFTP client to use.
type FTPClientConfig struct {
	// Client is the preferred client: "winscp" | "filezilla" | "" (auto-detect).
	Client string `yaml:"client"`
	// Path is the custom path to the executable; if empty, the default
	// locations are searched.
	Path string `yaml:"path"`
}

// Load reads the configuration file. If it does not exist, it is created
// with default content. Loading is best-effort: on any error a warning is
// logged and an empty config (all defaults) is returned, so callers never
// need to handle a missing or broken config file.
func Load() *Config {
	empty := &Config{Tags: map[string]string{}}

	data, err := os.ReadFile(paths.AppConfigFile())
	if os.IsNotExist(err) {
		if err := createDefault(); err != nil {
			slog.Warn("cannot create default ssx config", "err", err)
		}
		return empty
	}
	if err != nil {
		slog.Warn("cannot read ssx config", "err", err)
		return empty
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		slog.Warn("invalid ssx config", "err", err)
		return empty
	}
	if cfg.Tags == nil {
		cfg.Tags = map[string]string{}
	}
	return &cfg
}

func createDefault() error {
	if err := os.MkdirAll(paths.AppConfigDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(paths.AppConfigFile(), assets.DefaultConfig, 0o644)
}

// WriteSampleConfig writes the bundled sample config next to the user config
// so new options are discoverable after an update.
func WriteSampleConfig() error {
	if err := os.MkdirAll(paths.AppConfigDir(), 0o755); err != nil {
		return err
	}
	samplePath := paths.AppConfigSampleFile()
	return os.WriteFile(samplePath, assets.DefaultConfig, 0o644)
}
