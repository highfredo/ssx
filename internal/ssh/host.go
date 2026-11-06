// Package ssh parses the user's OpenSSH configuration and drives the native
// ssh binary to open interactive sessions and port-forwarding tunnels.
package ssh

import (
	"fmt"
	"os/exec"
	"strings"
)

// Tag is a host label with an optional display color.
type Tag struct {
	Name  string
	Color string // Inline color from "#>tags:name:color"; empty when not specified.
}

// HostConfig is the resolved configuration of a single Host entry.
type HostConfig struct {
	Title           string   // Display name: the Host line comment, or the alias when absent.
	Name            string   // The alias used after the Host keyword.
	Hostname        string   // Resolved HostName; falls back to Name when absent.
	User            string   // SSH user (User directive).
	Port            string   // SSH port (default: "22").
	Password        string   // Optional password (set via #>password:<value> comment).
	PasswordCommand string   // Shell command whose stdout is used as password (set via #>passwordCommand:<cmd> comment).
	IdentityFiles   []string // IdentityFile directives (can be multiple values).
	ProxyCommand    string   // ProxyCommand directive.
	ProxyJump       string   // ProxyJump directive.
	ConfigFile      string   // Path to the SSH config file where this host is defined; empty for ad-hoc hosts.
	ConfigLine      int      // 1-indexed line number of the Host directive in ConfigFile.
	Tags            []Tag
	Tunnels         []Tunnel
}

// IsAdHoc reports whether the host was built from a connection string
// (see ParseConnection) rather than from an SSH config file. Ad-hoc hosts are
// unknown to the ssh binary, so user and port must be passed explicitly.
func (h *HostConfig) IsAdHoc() bool {
	return h.ConfigFile == ""
}

// Target returns the host as "user@hostname", or just the hostname when no
// user is configured.
func (h *HostConfig) Target() string {
	if h.User == "" {
		return h.Hostname
	}
	return h.User + "@" + h.Hostname
}

// HasPassword reports whether password authentication is configured, either
// literally or through a command.
func (h *HostConfig) HasPassword() bool {
	return h.Password != "" || h.PasswordCommand != ""
}

// ResolvePassword returns the literal password or, failing that, the trimmed
// stdout of PasswordCommand. It returns "" when no password is configured.
func (h *HostConfig) ResolvePassword() (string, error) {
	if h.Password != "" {
		return h.Password, nil
	}
	if h.PasswordCommand == "" {
		return "", nil
	}
	out, err := exec.Command("sh", "-c", h.PasswordCommand).Output()
	if err != nil {
		return "", fmt.Errorf("passwordCommand %q: %w", h.PasswordCommand, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Hosts is an ordered collection of host configurations.
type Hosts []*HostConfig

// Get returns the host whose alias is name, or nil when not found.
func (hs Hosts) Get(name string) *HostConfig {
	for _, h := range hs {
		if h.Name == name {
			return h
		}
	}
	return nil
}
