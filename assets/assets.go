// Package assets expone los ficheros estáticos embebidos en el binario.
package assets

import _ "embed"

// DefaultConfig contiene el contenido del fichero de configuración de ejemplo.
//
//go:embed default_config.yaml
var DefaultConfig []byte
