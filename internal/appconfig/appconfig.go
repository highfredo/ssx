// Package appconfig gestiona la configuración de la aplicación ssx,
// leyendo de ~/.config/ssx/config.yaml con soporte XDG.
package appconfig

import (
	"log/slog"
	"os"

	"github.com/highfredo/ssx/assets"
	"github.com/highfredo/ssx/internal/paths"
	"gopkg.in/yaml.v3"
)

// Config contiene la configuración de ssx leída de ~/.config/ssx/config.yaml.
type Config struct {
	// LogLevel establece el nivel de logging: "debug", "info", "warn" o "error".
	// Si está vacío se usa "info".
	LogLevel string `yaml:"log_level"`
	// Theme personaliza la paleta de colores de la UI.
	Theme ThemeConfig `yaml:"theme"`
	// Tags define colores por defecto para cada etiqueta.
	Tags map[string]string `yaml:"tags"`
	// FTP configura el cliente SFTP preferido.
	FTP FTPClientConfig `yaml:"ftp"`
	// Keybindings permite personalizar los atajos de teclado.
	Keybindings KeybindingsConfig `yaml:"keybindings"`
}

// KeybindingsConfig agrupa todos los atajos de teclado personalizables.
// Cada campo es una cadena de teclas separadas por espacio (p.ej. "ctrl+t tab").
// Un campo vacío usa el valor por defecto.
type KeybindingsConfig struct {
	// Quit sale de la aplicación. Default: "ctrl+c".
	Quit string `yaml:"quit"`
	// Hosts agrupa los atajos de la pantalla principal de hosts.
	Hosts HostsKeybindings `yaml:"hosts"`
	// Tunnels agrupa los atajos de la pantalla de túneles.
	Tunnels TunnelsKeybindings `yaml:"tunnels"`
	// Navigation agrupa los atajos de navegación y scroll.
	Navigation NavigationKeybindings `yaml:"navigation"`
}

// HostsKeybindings contiene los atajos de la pantalla de hosts.
type HostsKeybindings struct {
	Connect      string `yaml:"connect"`       // Default: "enter"
	CopyKey      string `yaml:"copy_key"`      // Default: "ctrl+a"
	Tunnels      string `yaml:"tunnels"`       // Default: "ctrl+t tab"
	OpenTunnels  string `yaml:"open_tunnels"`  // Default: "ctrl+o"
	FTP          string `yaml:"ftp"`           // Default: "ctrl+x"
	Info         string `yaml:"info"`          // Default: "ctrl+g"
	OpenSSHDir   string `yaml:"open_ssh_dir"`  // Default: "ctrl+alt+p"
	OpenConfig   string `yaml:"open_config"`   // Default: "ctrl+alt+s"
	QuickConnect string `yaml:"quick_connect"` // Default: "ctrl+enter ctrl+j"
}

// TunnelsKeybindings contiene los atajos de la pantalla de túneles.
type TunnelsKeybindings struct {
	Toggle  string `yaml:"toggle"`  // Default: "enter"
	Browser string `yaml:"browser"` // Default: "ctrl+x"
	Back    string `yaml:"back"`    // Default: "esc tab"
}

// NavigationKeybindings contiene los atajos de navegación compartidos por
// todas las listas y vistas con scroll (hosts, tunnels, info).
type NavigationKeybindings struct {
	CursorUp     string `yaml:"cursor_up"`      // Default: "up"
	CursorDown   string `yaml:"cursor_down"`    // Default: "down"
	PrevPage     string `yaml:"prev_page"`      // Default: "pgup"
	NextPage     string `yaml:"next_page"`      // Default: "pgdown"
	GoToStart    string `yaml:"go_to_start"`    // Default: "home"
	GoToEnd      string `yaml:"go_to_end"`      // Default: "end"
	ShowFullHelp string `yaml:"show_full_help"` // Default: "f1"
	HalfPageUp   string `yaml:"half_page_up"`   // Default: "ctrl+u"
	HalfPageDown string `yaml:"half_page_down"` // Default: "ctrl+d"
	Back         string `yaml:"back"`           // Default: "tab esc"
}

// SlogLevel devuelve el slog.Level correspondiente al campo LogLevel.
// Valores válidos: "debug", "info", "warn", "error" (insensible a mayúsculas).
// Cualquier otro valor se trata como "info".
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

// ThemeConfig personaliza la paleta de colores de la interfaz de ssx.
// Cada campo acepta un color en formato hex (#RRGGBB), nombre CSS básico
// o número ANSI. Los campos vacíos mantienen el color por defecto.
type ThemeConfig struct {
	// Colores principales de la UI
	Primary    string `yaml:"primary"`    // acento / fondo de títulos    (default #7C3AED)
	Highlight  string `yaml:"highlight"`  // fila seleccionada (oscuro)    (default #1E1B4B)
	Success    string `yaml:"success"`    // estado abierto / OK           (default #10B981)
	Muted      string `yaml:"muted"`      // estado cerrado / ayuda        (default #6B7280)
	Danger     string `yaml:"danger"`     // errores                       (default #EF4444)
	Border     string `yaml:"border"`     // bordes de paneles
	DefaultTag string `yaml:"defaultTag"` //        (default #374151)
}

// FTPClientConfig configura qué cliente FTP/SFTP usar.
type FTPClientConfig struct {
	// Client es el cliente preferido: "winscp" | "filezilla" | "" (auto-detectar).
	Client string `yaml:"client"`
	// Path es la ruta personalizada al ejecutable; si está vacío se busca en
	// las ubicaciones por defecto.
	Path string `yaml:"path"`
}

// cached es la instancia cargada al inicio del programa.
var cached = load()

// Get devuelve la configuración cargada al inicio.
func Get() *Config {
	return cached
}

// load lee el fichero de configuración.
// Si no existe lo crea con contenido por defecto.
// Ante cualquier error devuelve una config vacía y lo loguea.
func load() *Config {
	empty := &Config{Tags: map[string]string{}}

	data, err := os.ReadFile(paths.AppConfigFile())
	if os.IsNotExist(err) {
		if err := createDefault(); err != nil {
			slog.Warn("no se pudo crear la configuración por defecto de ssx", "err", err)
		}
		return empty
	}
	if err != nil {
		slog.Warn("no se pudo leer la configuración de ssx", "err", err)
		return empty
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		slog.Warn("configuración de ssx inválida", "err", err)
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

func WriteSampleConfig() error {
	if err := os.MkdirAll(paths.AppConfigDir(), 0o755); err != nil {
		return err
	}
	samplePath := paths.AppConfigSampleFile()
	return os.WriteFile(samplePath, assets.DefaultConfig, 0o644)
}
