package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/paths"
	gossh "github.com/kevinburke/ssh_config"
)

// Tag representa una etiqueta de host con un color de visualización opcional.
type Tag struct {
	Name  string // nombre del tag
	Color string // color lipgloss (hex "#RRGGBB", nombre ANSI, etc.); vacío → color por defecto
}

type HostConfig struct {
	Title           string
	Name            string   // The alias used after the Host keyword or comment if the Host entry.
	Hostname        string   // Resolved HostName; always set (falls back to the alias/Name if absent).
	User            string   // SSH user (User directive).
	Port            string   // SSH port (default: "22").
	Password        string   // Optional password (set via #>password:<value> comment).
	PasswordCommand string   // Shell command whose stdout is used as password (set via #>passwordCommand:<cmd> comment).
	IdentityFiles   []string // IdentityFile directives (can be multiple values).
	ProxyCommand    string   // ProxyCommand directive.
	ProxyJump       string   // ProxyJump directive.
	Tags            []Tag
	Tunnels         []Tunnel
}

type TunnelType int

const (
	TunnelTypeLocal   TunnelType = iota // -L: local port → remote host:port
	TunnelTypeRemote                    // -R: remote port → local host:port
	TunnelTypeDynamic                   // -D: SOCKS5 proxy on local port
)

type Tunnel struct {
	Type        TunnelType
	Host        string
	LocalPort   string // Bind port on the local machine.
	RemoteHost  string // Destination host (empty for dynamic).
	RemotePort  string // Destination port (empty for dynamic).
	Description string // Comment of the tunnel (if any).
}

var cachedHost []*HostConfig

// LoadConfig reads ~/.ssh/config and returns all non-wildcard host entries,
// including those declared in Include files.
func LoadConfig() ([]*HostConfig, error) {
	path := paths.SSHConfigFile()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	cfg, err := gossh.Decode(f)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)

	cachedHost = collectHosts(cfg, cfg, seen)
	return cachedHost, nil
}

func GetHosts() []*HostConfig {
	return cachedHost
}

func GetHost(name string) *HostConfig {
	for _, host := range cachedHost {
		if host.Name == name {
			return host
		}
	}
	return nil
}

// collectHosts walks cfg.Hosts and descends into any Include nodes, using
// rootCfg.Get() for all directive lookups (it already traverses includes).
func collectHosts(cfg *gossh.Config, rootCfg *gossh.Config, seen map[string]bool) []*HostConfig {
	// Wildcard nodes from both the current file and the root config act as
	// fallback values for magic comments (first-match semantics, same as SSH).
	fallbackNodes := wildcardNodes(cfg)
	if rootCfg != cfg {
		fallbackNodes = append(fallbackNodes, wildcardNodes(rootCfg)...)
	}

	var hosts []*HostConfig
	for _, host := range cfg.Hosts {
		// Recurse into Include nodes before processing this host's own patterns,
		// so that first-occurrence ordering (SSH semantics) is respected.
		for _, node := range host.Nodes {
			if inc, ok := node.(*gossh.Include); ok {
				for _, subCfg := range subConfigsOf(inc) {
					hosts = append(hosts, collectHosts(subCfg, rootCfg, seen)...)
				}
			}
		}

		// Count distinct non-wildcard aliases on this Host line so we can
		// disambiguate the Title when multiple aliases share the same comment.
		nonWildcardCount := 0
		for _, p := range host.Patterns {
			if !strings.ContainsAny(p.String(), "*?") {
				nonWildcardCount++
			}
		}
		comment := strings.TrimSpace(host.EOLComment)

		for _, pattern := range host.Patterns {
			alias := pattern.String()
			if strings.ContainsAny(alias, "*?") || seen[alias] {
				continue
			}
			seen[alias] = true

			// Combine the specific host nodes with wildcard fallback nodes so
			// that magic comments are inherited (first-match: specific wins).
			nodes := append(host.Nodes, fallbackNodes...)

			if strings.EqualFold(extractComment(nodes, ">hidden:"), "true") {
				continue
			}

			var title string
			switch {
			case comment != "" && nonWildcardCount > 1:
				title = fmt.Sprintf("%s (%s)", comment, alias)
			case comment != "":
				title = comment
			default:
				title = alias
			}

			tunnels := collectTunnelsFromNodes(host.Nodes)
			for i := range tunnels {
				tunnels[i].Host = alias
			}

			hostname, _ := rootCfg.Get(alias, "HostName")
			if hostname == "" {
				hostname = alias
			}
			port, _ := rootCfg.Get(alias, "Port")
			if port == "" {
				port = "22"
			}
			user, _ := rootCfg.Get(alias, "User")
			identityFiles, _ := rootCfg.GetAll(alias, "IdentityFile")
			proxyCommand, _ := rootCfg.Get(alias, "ProxyCommand")
			proxyJump, _ := rootCfg.Get(alias, "ProxyJump")

			hosts = append(hosts, &HostConfig{
				Title:           title,
				Name:            alias,
				Hostname:        hostname,
				User:            user,
				Port:            port,
				Password:        extractPassword(nodes),
				PasswordCommand: extractPasswordCommand(nodes),
				IdentityFiles:   filterSSHNone(identityFiles),
				ProxyCommand:    normalizeSSHValue(proxyCommand),
				ProxyJump:       normalizeSSHValue(proxyJump),
				Tags:            extractTags(nodes),
				Tunnels:         tunnels,
			})
		}
	}
	return hosts
}

// wildcardNodes collects all nodes that belong to wildcard Host patterns
// (patterns containing * or ?) in cfg. These are used as fallback values
// for magic comments, mirroring how SSH config inherits directives from Host *.
func wildcardNodes(cfg *gossh.Config) []gossh.Node {
	var nodes []gossh.Node
	for _, host := range cfg.Hosts {
		for _, p := range host.Patterns {
			if strings.ContainsAny(p.String(), "*?") {
				nodes = append(nodes, host.Nodes...)
				break
			}
		}
	}
	return nodes
}

func normalizeSSHValue(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), "none") {
		return ""
	}
	return v
}

func filterSSHNone(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if normalized := normalizeSSHValue(value); normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}

// subConfigsOf extracts the raw directive paths from an Include node's string
// representation and re-parses each matched file into a *gossh.Config.
func subConfigsOf(inc *gossh.Include) []*gossh.Config {
	// inc.String() → e.g. "  Include ~/.ssh/conf.d/*.conf"
	parts := strings.Fields(inc.String())
	if len(parts) < 2 {
		return nil
	}
	// Skip "Include" keyword and optional "=" separator.
	directives := parts[1:]
	if len(directives) > 0 && directives[0] == "=" {
		directives = directives[1:]
	}

	var configs []*gossh.Config
	for _, directive := range directives {
		paths, err := resolveIncludePath(directive)
		if err != nil {
			continue
		}
		for _, p := range paths {
			subCfg, err := decodeFile(p)
			if err != nil {
				continue
			}
			configs = append(configs, subCfg)
		}
	}
	return configs
}

// resolveIncludePath expands a single Include directive pattern to matching
// file paths, mirroring the resolution rules of the ssh_config library.
func resolveIncludePath(directive string) ([]string, error) {
	var base string
	switch {
	case filepath.IsAbs(directive):
		base = directive
	case strings.HasPrefix(directive, "~/"):
		base = filepath.Join(paths.Home(), directive[2:])
	default:
		base = filepath.Join(paths.SSHDir(), directive)
	}
	return filepath.Glob(base)
}

func decodeFile(path string) (*gossh.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return gossh.Decode(f)
}

// collectTunnelsFromNodes extracts tunnel directives from a slice of SSH config
// nodes, recursively descending into Include nodes.
func collectTunnelsFromNodes(nodes []gossh.Node) []Tunnel {
	var tunnels []Tunnel
	for _, node := range nodes {
		switch n := node.(type) {
		case *gossh.KV:
			desc := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(n.Comment), "#"))
			switch strings.ToLower(n.Key) {
			case "localforward":
				if t, ok := parseForward(TunnelTypeLocal, n.Value, desc); ok {
					tunnels = append(tunnels, t)
				}
			case "remoteforward":
				if t, ok := parseForward(TunnelTypeRemote, n.Value, desc); ok {
					tunnels = append(tunnels, t)
				}
			case "dynamicforward":
				if t, ok := parseDynamicForward(n.Value, desc); ok {
					tunnels = append(tunnels, t)
				}
			}
		case *gossh.Include:
			// Include inside a Host block: collect tunnel directives from the
			// implicit Host * blocks of each included file.
			for _, subCfg := range subConfigsOf(n) {
				for _, subHost := range subCfg.Hosts {
					for _, p := range subHost.Patterns {
						if p.String() == "*" {
							tunnels = append(tunnels, collectTunnelsFromNodes(subHost.Nodes)...)
						}
					}
				}
			}
		}
	}
	return tunnels
}

// parseForward parses "LocalForward" / "RemoteForward" specs of the form:
// [bindaddr:]localport remotehost:remoteport
func parseForward(tunnelType TunnelType, spec, desc string) (Tunnel, bool) {
	parts := strings.Fields(spec)
	if len(parts) < 2 {
		return Tunnel{}, false
	}
	localPort := parts[0]
	if idx := strings.LastIndex(localPort, ":"); idx >= 0 {
		localPort = localPort[idx+1:]
	}
	remoteHost, remotePort, err := splitHostPort(parts[1])
	if err != nil {
		return Tunnel{}, false
	}
	return Tunnel{
		Type:        tunnelType,
		LocalPort:   localPort,
		RemoteHost:  remoteHost,
		RemotePort:  remotePort,
		Description: desc,
	}, true
}

// parseDynamicForward parses "DynamicForward" specs of the form: [bindaddr:]port
func parseDynamicForward(spec, desc string) (Tunnel, bool) {
	spec = strings.TrimSpace(spec)
	if idx := strings.LastIndex(spec, ":"); idx >= 0 {
		spec = spec[idx+1:]
	}
	if spec == "" {
		return Tunnel{}, false
	}
	return Tunnel{
		Type:        TunnelTypeDynamic,
		LocalPort:   spec,
		Description: desc,
	}, true
}

// extractPassword scans SSH config nodes for a standalone comment of the form
// #>password:<value> and returns the value, or empty string if not found.
func extractPassword(nodes []gossh.Node) string {
	return extractComment(nodes, ">password:")
}

// extractPasswordCommand scans SSH config nodes for a standalone comment of the form
// #>passwordCommand:<cmd> and returns the command, or empty string if not found.
func extractPasswordCommand(nodes []gossh.Node) string {
	return extractComment(nodes, ">passwordCommand:")
}

// extractComment is the shared implementation for extractPassword and extractPasswordCommand.
func extractComment(nodes []gossh.Node, prefix string) string {
	for _, node := range nodes {
		var comment string
		switch n := node.(type) {
		case *gossh.Empty:
			comment = strings.TrimSpace(n.Comment)
		case *gossh.KV:
			comment = strings.TrimSpace(n.Comment)
		}
		if after, ok := strings.CutPrefix(comment, prefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

// extractTags scans all SSH config nodes for #>tags: comments and merges the
// results. This allows wildcard Host blocks (e.g. Host *) to contribute tags
// that are inherited by specific hosts. When the same tag name appears more
// than once (e.g. the host overrides the wildcard), the first occurrence wins
// (first-match semantics), preserving the host-specific color if any.
//
// Each tag entry has the form: name[:color], name[:color], ...
// If a tag has no inline color, the YAML app config tagColors map is used as
// fallback.
func extractTags(nodes []gossh.Node) []Tag {
	seen := make(map[string]bool)
	var tags []Tag

	for _, node := range nodes {
		var comment string
		switch n := node.(type) {
		case *gossh.Empty:
			comment = strings.TrimSpace(n.Comment)
		case *gossh.KV:
			comment = strings.TrimSpace(n.Comment)
		}
		after, ok := strings.CutPrefix(comment, ">tags:")
		if !ok {
			continue
		}
		for _, raw := range strings.Split(strings.TrimSpace(after), ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			name, color, _ := strings.Cut(raw, ":")
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			color = strings.TrimSpace(color)
			if color == "" {
				color = appconfig.Get().Tags[name]
			}
			tags = append(tags, Tag{Name: name, Color: color})
		}
	}
	return tags
}

// splitHostPort handles both IPv6 ([addr]:port) and regular (host:port) formats.
func splitHostPort(s string) (host, port string, err error) {
	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end < 0 {
			return "", "", fmt.Errorf("invalid IPv6 address: %s", s)
		}
		host = s[1:end]
		rest := s[end+1:]
		if !strings.HasPrefix(rest, ":") {
			return "", "", fmt.Errorf("missing port in address: %s", s)
		}
		return host, rest[1:], nil
	}
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("missing port in address: %s", s)
	}
	return s[:idx], s[idx+1:], nil
}
