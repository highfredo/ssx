package ssh

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/highfredo/ssx/internal/paths"
	gossh "github.com/kevinburke/ssh_config"
)

// LoadConfig reads ~/.ssh/config and returns all non-wildcard host entries,
// including those declared in Include files. A missing file yields no hosts
// and no error.
func LoadConfig() (Hosts, error) {
	path := paths.SSHConfigFile()
	cfg, err := decodeFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read ssh config %s: %w", path, err)
	}
	return newHostCollector(cfg).collect(cfg, path), nil
}

// hostCollector walks a parsed ssh_config tree and builds HostConfig values.
// Directive lookups always go through root, which already traverses includes
// with SSH's first-match semantics.
type hostCollector struct {
	root *gossh.Config
	seen map[string]bool // aliases already emitted; SSH keeps the first occurrence
}

func newHostCollector(root *gossh.Config) *hostCollector {
	return &hostCollector{root: root, seen: make(map[string]bool)}
}

// collect returns the hosts declared in cfg (located at configPath), descending
// into Include nodes in document order so that first-occurrence wins.
func (c *hostCollector) collect(cfg *gossh.Config, configPath string) []*HostConfig {
	// Wildcard blocks from the current file and from the root act as fallback
	// sources for magic comments, mirroring how SSH inherits from "Host *".
	fallback := wildcardNodes(cfg)
	if cfg != c.root {
		fallback = append(fallback, wildcardNodes(c.root)...)
	}

	var hosts []*HostConfig
	lastLine := 0
	for _, block := range cfg.Hosts {
		var hostLine int
		hostLine, lastLine = blockLines(block, lastLine)

		for _, inc := range includeNodes(block.Nodes) {
			for _, sub := range includedConfigs(inc) {
				hosts = append(hosts, c.collect(sub.cfg, sub.path)...)
			}
		}

		hosts = append(hosts, c.hostsFromBlock(block, fallback, configPath, hostLine)...)
	}
	return hosts
}

// hostsFromBlock builds one HostConfig per concrete (non-wildcard) alias of a
// Host block, skipping aliases already seen and blocks marked #>hidden:true.
func (c *hostCollector) hostsFromBlock(block *gossh.Host, fallback []gossh.Node, configPath string, line int) []*HostConfig {
	aliases := concreteAliases(block)
	// Specific nodes first so that they win over wildcard fallbacks.
	nodes := append(slices.Clone(block.Nodes), fallback...)
	comment := strings.TrimSpace(block.EOLComment)

	var hosts []*HostConfig
	for _, alias := range aliases {
		if c.seen[alias] {
			continue
		}
		c.seen[alias] = true
		if strings.EqualFold(magicValue(nodes, "hidden"), "true") {
			continue
		}
		hosts = append(hosts, c.buildHost(alias, hostTitle(comment, alias, len(aliases)), block.Nodes, nodes, configPath, line))
	}
	return hosts
}

// buildHost resolves every directive for alias. ownNodes are the block's own
// nodes (used for tunnels, which must not be inherited), while nodes also
// include wildcard fallbacks (used for magic comments).
func (c *hostCollector) buildHost(alias, title string, ownNodes, nodes []gossh.Node, configPath string, line int) *HostConfig {
	tunnels := collectTunnels(ownNodes)
	for i := range tunnels {
		tunnels[i].Host = alias
	}
	identityFiles, _ := c.root.GetAll(alias, "IdentityFile")

	return &HostConfig{
		Title:           title,
		Name:            alias,
		Hostname:        c.get(alias, "HostName", alias),
		User:            c.get(alias, "User", ""),
		Port:            c.get(alias, "Port", DefaultPort),
		Password:        magicValue(nodes, "password"),
		PasswordCommand: magicValue(nodes, "passwordCommand"),
		IdentityFiles:   filterSSHNone(identityFiles),
		ProxyCommand:    normalizeSSHValue(c.get(alias, "ProxyCommand", "")),
		ProxyJump:       normalizeSSHValue(c.get(alias, "ProxyJump", "")),
		ConfigFile:      configPath,
		ConfigLine:      line,
		Tags:            extractTags(nodes),
		Tunnels:         tunnels,
	}
}

// get returns the effective value of key for alias, or def when unset.
func (c *hostCollector) get(alias, key, def string) string {
	v, _ := c.root.Get(alias, key)
	if v == "" {
		return def
	}
	return v
}

// blockLines estimates the 1-indexed line of a Host directive from the
// positions of its child nodes, since ssh_config does not expose it directly.
// It returns the Host line and the last line consumed by the block.
func blockLines(block *gossh.Host, prevLast int) (hostLine, lastLine int) {
	hostLine = prevLast + 1
	if len(block.Nodes) == 0 {
		return hostLine, hostLine
	}
	lastLine = prevLast
	if pos := block.Nodes[0].Pos(); pos.Line > 1 {
		hostLine = pos.Line - 1
	}
	if end := block.Nodes[len(block.Nodes)-1].Pos(); end.Line > lastLine {
		lastLine = end.Line
	}
	return hostLine, lastLine
}

// hostTitle derives the display title: the Host line comment, disambiguated
// with the alias when several aliases share the same comment.
func hostTitle(comment, alias string, aliasCount int) string {
	switch {
	case comment != "" && aliasCount > 1:
		return fmt.Sprintf("%s (%s)", comment, alias)
	case comment != "":
		return comment
	default:
		return alias
	}
}

// concreteAliases returns the block patterns that are not wildcards.
func concreteAliases(block *gossh.Host) []string {
	var aliases []string
	for _, p := range block.Patterns {
		if !isWildcard(p.String()) {
			aliases = append(aliases, p.String())
		}
	}
	return aliases
}

func isWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?")
}

// wildcardNodes collects the nodes of every wildcard Host block in cfg.
func wildcardNodes(cfg *gossh.Config) []gossh.Node {
	var nodes []gossh.Node
	for _, block := range cfg.Hosts {
		if slices.ContainsFunc(block.Patterns, func(p *gossh.Pattern) bool { return isWildcard(p.String()) }) {
			nodes = append(nodes, block.Nodes...)
		}
	}
	return nodes
}

// normalizeSSHValue maps the SSH keyword "none" to the empty string.
func normalizeSSHValue(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), "none") {
		return ""
	}
	return v
}

func filterSSHNone(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if normalized := normalizeSSHValue(v); normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}
