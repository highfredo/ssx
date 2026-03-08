// Package appconfig gestiona la configuración de la aplicación ssx,
// leyendo de ~/.config/ssx/config.yaml con soporte XDG.
package appconfig

import (
	"log/slog"
	"os"

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

const defaultContent = `# ssx configuration file
# https://github.com/highfredo/ssx

# Log level: "debug", "info", "warn" or "error" (default: "info").
# Logs are written to /tmp/ssx.log.
log_level: "info"

# UI theme: customize the color palette.
# Each value accepts a hex color (#RRGGBB), basic CSS name, or ANSI number.
# Leave a field empty (or remove it) to keep the default color.
#
# Defaults (purple/indigo palette):
# theme:
#   primary:    "#7C3AED"   # accent / title background
#   highlight:  "#1E1B4B"   # selected row background (dark)
#   success:    "#10B981"   # open state / success messages
#   muted:      "#6B7280"   # closed state / help text
#   danger:     "#EF4444"   # errors
#   border:     "#374151"   # panel borders
# 	defaultTag: "#312E81"   # default tag color (used when no color is set in ~/.ssh/config)
theme: {}

# FTP/SFTP client configuration.
#
# client: preferred client to use: "winscp" or "filezilla"
#         leave empty to auto-detect (WinSCP first, then FileZilla)
# path:   optional custom path to the executable
#         leave empty to search in default locations
#
# Example:
# ftp:
#   client: filezilla
#   path: /usr/bin/filezilla
ftp:
  client: ""
  path: ""

# Tag colors: assign a background color (hex or CSS name) to each tag.
# Colors defined here are used as fallback; colors in ~/.ssh/config take priority.
#
# Example:
# tags:
#   personal: "#10B981"
#   work: "#3B82F6"
#   production: "#EF4444"
tags: {}
`

func createDefault() error {
	if err := os.MkdirAll(paths.AppConfigDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(paths.AppConfigFile(), []byte(defaultContent), 0o644)
}
