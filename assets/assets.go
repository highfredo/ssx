// Package assets exposes the static files embedded in the binary.
package assets

import _ "embed"

// DefaultConfig holds the contents of the sample configuration file.
//
//go:embed default_config.yaml
var DefaultConfig []byte
