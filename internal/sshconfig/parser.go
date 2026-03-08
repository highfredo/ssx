// Package sshconfig parses and represents SSH host configuration from ~/.ssh/config.
// It extracts host entries, their effective connection parameters, and any
// port-forwarding tunnels (LocalForward, RemoteForward, DynamicForward).
// It also parses optional host tags declared as comments: "#>tags: a, b".
package sshconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gossh "github.com/kevinburke/ssh_config"
)

// TunnelType enumerates the supported SSH port-forwarding modes.
type TunnelType string

const (
	TunnelTypeLocal   TunnelType = "local"   // -L: local port → remote host:port
	TunnelTypeRemote  TunnelType = "remote"  // -R: remote port → local host:port
	TunnelTypeDynamic TunnelType = "dynamic" // -D: SOCKS5 proxy on local port
)

// Tunnel represents a single SSH port-forwarding rule parsed from ~/.ssh/config.
type Tunnel struct {
	Type       TunnelType
	LocalPort  string // Bind port on the local machine.
	RemoteHost string // Destination host (empty for dynamic).
	RemotePort string // Destination port (empty for dynamic).
}

// ID returns a stable, unique key for this tunnel scoped to a host alias.
// Used as map keys in the tunnel manager.
func (t Tunnel) ID(hostName string) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s",
		hostName, t.Type, t.LocalPort, t.RemoteHost, t.RemotePort)
}

// DisplaySpec returns a human-readable forwarding specification.
func (t Tunnel) DisplaySpec() string {
	switch t.Type {
	case TunnelTypeDynamic:
		return fmt.Sprintf("SOCKS5 on :%s", t.LocalPort)
	default:
		return fmt.Sprintf(":%s → %s:%s", t.LocalPort, t.RemoteHost, t.RemotePort)
	}
}

// Host represents a single SSH host entry from ~/.ssh/config.
type Host struct {
	Name     string // The alias used after the Host keyword.
	Hostname string // Resolved HostName (or alias if absent).
	User     string // SSH user (User directive).
	Port     string // SSH port (default: "22").
	Tags     []string
	Tunnels  []Tunnel
}

// ParseConfig reads and parses ~/.ssh/config, returning all non-wildcard host entries.
// If the file does not exist an empty slice is returned without error.
func ParseConfig() ([]*Host, error) {
	configPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open ssh config %s: %w", configPath, err)
	}

	cfg, err := gossh.Decode(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("parse ssh config: %w", err)
	}
	hostTags := parseHostTags(string(content))

	var hosts []*Host
	for _, h := range cfg.Hosts {
		if len(h.Patterns) == 0 {
			continue
		}
		name := h.Patterns[0].String()
		// Skip wildcard / catch-all patterns.
		if strings.ContainsAny(name, "*?") {
			continue
		}

		hostname, _ := cfg.Get(name, "HostName")
		if hostname == "" {
			hostname = name
		}
		user, _ := cfg.Get(name, "User")
		port, _ := cfg.Get(name, "Port")
		if port == "" {
			port = "22"
		}

		hosts = append(hosts, &Host{
			Name:     name,
			Hostname: hostname,
			User:     user,
			Port:     port,
			Tags:     hostTags[name],
			Tunnels:  parseTunnels(cfg, name),
		})
	}
	return hosts, nil
}

// parseHostTags extracts "#>tags: ..." comments from Host blocks.
func parseHostTags(configText string) map[string][]string {
	tagsByAlias := make(map[string][]string)
	var currentAliases []string

	scanner := bufio.NewScanner(strings.NewReader(configText))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) > 0 {
			switch strings.ToLower(fields[0]) {
			case "host":
				currentAliases = nil
				if len(fields) > 1 {
					currentAliases = append(currentAliases, fields[1:]...)
				}
				continue
			case "match":
				currentAliases = nil
			}
		}

		if len(currentAliases) == 0 || !strings.HasPrefix(line, "#>") {
			continue
		}
		anno := strings.TrimSpace(strings.TrimPrefix(line, "#>"))
		if !strings.HasPrefix(strings.ToLower(anno), "tags:") {
			continue
		}

		tagText := strings.TrimSpace(anno[len("tags:"):])
		if tagText == "" {
			continue
		}
		tags := strings.Split(tagText, ",")
		for _, alias := range currentAliases {
			if strings.ContainsAny(alias, "*?") || strings.HasPrefix(alias, "!") {
				continue
			}
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag == "" {
					continue
				}
				tagsByAlias[alias] = appendIfMissing(tagsByAlias[alias], tag)
			}
		}
	}
	return tagsByAlias
}

func appendIfMissing(items []string, v string) []string {
	for _, item := range items {
		if item == v {
			return items
		}
	}
	return append(items, v)
}

// parseTunnels extracts all configured tunnels for the given host alias.
func parseTunnels(cfg *gossh.Config, alias string) []Tunnel {
	var tunnels []Tunnel

	if localFwds, _ := cfg.GetAll(alias, "LocalForward"); len(localFwds) > 0 {
		for _, fwd := range localFwds {
			if t, ok := parseForward(TunnelTypeLocal, fwd); ok {
				tunnels = append(tunnels, t)
			}
		}
	}

	if remoteFwds, _ := cfg.GetAll(alias, "RemoteForward"); len(remoteFwds) > 0 {
		for _, fwd := range remoteFwds {
			if t, ok := parseForward(TunnelTypeRemote, fwd); ok {
				tunnels = append(tunnels, t)
			}
		}
	}

	if dynFwds, _ := cfg.GetAll(alias, "DynamicForward"); len(dynFwds) > 0 {
		for _, p := range dynFwds {
			p = strings.TrimSpace(p)
			// Strip optional bind address (e.g. "127.0.0.1:1080" → "1080").
			if idx := strings.LastIndex(p, ":"); idx >= 0 {
				p = p[idx+1:]
			}
			if p != "" {
				tunnels = append(tunnels, Tunnel{
					Type:      TunnelTypeDynamic,
					LocalPort: p,
				})
			}
		}
	}

	return tunnels
}

// parseForward parses a LocalForward/RemoteForward spec:
//
//	"[bindaddr:]localport remotehost:remoteport"
func parseForward(typ TunnelType, spec string) (Tunnel, bool) {
	parts := strings.Fields(spec)
	if len(parts) < 2 {
		return Tunnel{}, false
	}

	// First token: optional bind address + local port.
	localSpec := parts[0]
	localPort := localSpec
	if idx := strings.LastIndex(localSpec, ":"); idx >= 0 {
		localPort = localSpec[idx+1:]
	}

	// Second token: remote address (host:port or [ipv6]:port).
	remoteHost, remotePort, ok := splitHostPort(parts[1])
	if !ok {
		return Tunnel{}, false
	}

	return Tunnel{
		Type:       typ,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
	}, true
}

// splitHostPort splits "host:port" or "[ipv6addr]:port" into its components.
func splitHostPort(s string) (host, port string, ok bool) {
	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end < 0 {
			return "", "", false
		}
		host = s[1:end]
		rest := s[end+1:]
		if !strings.HasPrefix(rest, ":") {
			return "", "", false
		}
		return host, rest[1:], true
	}
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}
