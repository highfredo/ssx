package ssh

import (
	"strings"

	gossh "github.com/kevinburke/ssh_config"
)

// collectTunnels extracts LocalForward, RemoteForward and DynamicForward
// directives from nodes. For Include nodes inside a Host block, it collects
// the directives from the implicit "Host *" blocks of each included file.
func collectTunnels(nodes []gossh.Node) []Tunnel {
	var tunnels []Tunnel
	for _, node := range nodes {
		switch n := node.(type) {
		case *gossh.KV:
			if t, ok := parseForwardDirective(n); ok {
				tunnels = append(tunnels, t)
			}
		case *gossh.Include:
			for _, sub := range includedConfigs(n) {
				for _, block := range sub.cfg.Hosts {
					if blockMatchesAll(block) {
						tunnels = append(tunnels, collectTunnels(block.Nodes)...)
					}
				}
			}
		}
	}
	return tunnels
}

func blockMatchesAll(block *gossh.Host) bool {
	for _, p := range block.Patterns {
		if p.String() == "*" {
			return true
		}
	}
	return false
}

// parseForwardDirective converts a *Forward directive into a Tunnel; its
// trailing comment becomes the tunnel description.
func parseForwardDirective(kv *gossh.KV) (Tunnel, bool) {
	desc := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(kv.Comment), "#"))
	switch strings.ToLower(kv.Key) {
	case "localforward":
		return parseForward(TunnelTypeLocal, kv.Value, desc)
	case "remoteforward":
		return parseForward(TunnelTypeRemote, kv.Value, desc)
	case "dynamicforward":
		return parseDynamicForward(kv.Value, desc)
	}
	return Tunnel{}, false
}

// parseForward parses "LocalForward" / "RemoteForward" specs of the form:
// [bindaddr:]localport remotehost:remoteport
func parseForward(tunnelType TunnelType, spec, desc string) (Tunnel, bool) {
	parts := strings.Fields(spec)
	if len(parts) < 2 {
		return Tunnel{}, false
	}
	localPort := stripBindAddress(parts[0])
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
	port := stripBindAddress(strings.TrimSpace(spec))
	if port == "" {
		return Tunnel{}, false
	}
	return Tunnel{
		Type:        TunnelTypeDynamic,
		LocalPort:   port,
		Description: desc,
	}, true
}

// stripBindAddress returns the port of a "[bindaddr:]port" spec.
func stripBindAddress(spec string) string {
	if idx := strings.LastIndex(spec, ":"); idx >= 0 {
		return spec[idx+1:]
	}
	return spec
}
