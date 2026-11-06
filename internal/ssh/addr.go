package ssh

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// validHostRe allows hostnames, FQDNs, plain IPv4 and IPv6 addresses
// (brackets and colons are kept after net.SplitHostPort strips the brackets).
var validHostRe = regexp.MustCompile(`^[a-zA-Z0-9._:\-\[\]%]+$`)

// validUserRe follows the POSIX portable filename character set for usernames.
var validUserRe = regexp.MustCompile(`^[a-zA-Z0-9._\-]+$`)

// DefaultPort is the SSH port assumed when none is configured.
const DefaultPort = "22"

// ParseConnection parses a connection string of the form
// [user[:password]@]host[:port] and returns an ad-hoc HostConfig, or nil when
// the string is not a valid connection target.
func ParseConnection(s string) *HostConfig {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var user, pass string
	if i := strings.LastIndex(s, "@"); i != -1 {
		user, pass, _ = strings.Cut(s[:i], ":")
		s = s[i+1:]
	}

	host, port, err := net.SplitHostPort(s)
	if err != nil {
		host = s
	}
	if port == "" {
		port = DefaultPort
	}

	if !validHostRe.MatchString(host) || !isValidPort(port) {
		return nil
	}
	if user != "" && !validUserRe.MatchString(user) {
		return nil
	}

	return &HostConfig{
		Name:     host,
		Hostname: host,
		User:     user,
		Port:     port,
		Password: pass,
	}
}

func isValidPort(s string) bool {
	p, err := strconv.Atoi(s)
	return err == nil && p >= 1 && p <= 65535
}

// isValidHost accepts hostnames and IPv4/IPv6 literals, rejecting bare ports.
func isValidHost(s string) bool {
	if s == "" || isValidPort(s) || !validHostRe.MatchString(s) {
		return false
	}
	if strings.Contains(s, ":") {
		return net.ParseIP(s) != nil
	}
	return true
}

// splitHostPort splits "host:port" and "[ipv6]:port" into their parts.
func splitHostPort(s string) (host, port string, err error) {
	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end < 0 {
			return "", "", fmt.Errorf("invalid IPv6 address: %s", s)
		}
		rest := s[end+1:]
		if !strings.HasPrefix(rest, ":") {
			return "", "", fmt.Errorf("missing port in address: %s", s)
		}
		return s[1:end], rest[1:], nil
	}
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("missing port in address: %s", s)
	}
	return s[:idx], s[idx+1:], nil
}
