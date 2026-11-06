package ssh

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/highfredo/ssx/internal/paths"
	gossh "github.com/kevinburke/ssh_config"
)

// includedConfig is a parsed file referenced by an Include directive.
type includedConfig struct {
	path string
	cfg  *gossh.Config
}

// includeNodes returns the Include nodes among nodes.
func includeNodes(nodes []gossh.Node) []*gossh.Include {
	var incs []*gossh.Include
	for _, n := range nodes {
		if inc, ok := n.(*gossh.Include); ok {
			incs = append(incs, inc)
		}
	}
	return incs
}

// includedConfigs resolves every path pattern of an Include directive and
// parses each matched file. Unreadable files are skipped.
func includedConfigs(inc *gossh.Include) []includedConfig {
	// ssh_config does not expose the raw directive, so recover it from the
	// node's string form: "  Include [=] <pattern>...".
	fields := strings.Fields(inc.String())
	if len(fields) < 2 {
		return nil
	}
	patterns := fields[1:]
	if patterns[0] == "=" {
		patterns = patterns[1:]
	}

	var configs []includedConfig
	for _, pattern := range patterns {
		matches, err := filepath.Glob(resolveIncludePath(pattern))
		if err != nil {
			continue
		}
		for _, p := range matches {
			cfg, err := decodeFile(p)
			if err != nil {
				continue
			}
			configs = append(configs, includedConfig{path: p, cfg: cfg})
		}
	}
	return configs
}

// resolveIncludePath expands an Include pattern following OpenSSH rules:
// absolute paths and "~/" are used as-is, anything else is relative to ~/.ssh.
func resolveIncludePath(pattern string) string {
	switch {
	case filepath.IsAbs(pattern):
		return pattern
	case strings.HasPrefix(pattern, "~/"):
		return filepath.Join(paths.Home(), pattern[2:])
	default:
		return filepath.Join(paths.SSHDir(), pattern)
	}
}

func decodeFile(path string) (*gossh.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return gossh.Decode(f)
}
