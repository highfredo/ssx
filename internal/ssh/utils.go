package ssh

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func Shell(config *HostConfig) *exec.Cmd {
	return Run(config,
		"-o", "ClearAllForwardings=yes",
	)
}

// Run builds an interactive ssh command (no remote command).
// Extra flags in args are prepended before the common options.
func Run(config *HostConfig, args ...string) *exec.Cmd {
	return RunRemote(config, args, []string{})
}

func RunRemote(config *HostConfig, args []string, command []string) *exec.Cmd {
	slog.Debug("running", "ssh", args, "command", command)
	runArgs := append(args, buildSSHArgs(config)...)

	cmd := exec.Command("ssh", runArgs...)

	if config.Password != "" {
		ConfigurePassword(cmd, config.Password)
	} else if config.PasswordCommand != "" {
		ConfigurePasswordCommand(cmd, config.PasswordCommand)
	} else {
		cmd.Args = append(cmd.Args, "-o", "BatchMode=yes")
	}

	cmd.Args = append(cmd.Args, config.Hostname)
	cmd.Args = append(cmd.Args, command...)

	slog.Debug("SSH: ", "command", cmd.Args, "config", config)

	return cmd
}

// CopyId builds an ssh command that installs the local public key on the
// remote host in a single connection, avoiding the ssh-copy-id double-connect
// hang. It reads the public key locally and embeds it in the remote command.
func CopyId(config *HostConfig) (*exec.Cmd, error) {
	pubKeyPath, err := FindPublicKey(config)
	if err != nil {
		return nil, err
	}
	pubKeyData, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key %s: %w", pubKeyPath, err)
	}

	key := strings.TrimSpace(string(pubKeyData))
	// SSH public keys never contain single quotes, so single-quoting is safe.
	// %%s becomes a literal %s for the printf format specifier.
	remoteCmd := fmt.Sprintf(
		`umask 077 && mkdir -p ~/.ssh && (grep -qxF '%s' ~/.ssh/authorized_keys 2>/dev/null || printf '%%s\n' '%s' >> ~/.ssh/authorized_keys) && chmod 600 ~/.ssh/authorized_keys && chmod 700 ~/.ssh`,
		key, key,
	)

	sshArgs := []string{"-o", "ClearAllForwardings=yes"}

	return RunRemote(config, sshArgs, []string{remoteCmd}), nil
}

// buildSSHArgs returns the common SSH flags derived from config
// (user, port, identity files, proxy options, password prompts).
func buildSSHArgs(config *HostConfig) []string {
	var args []string
	if config.User != "" {
		args = append(args, "-l", config.User)
	}
	if config.Port != "" {
		args = append(args, "-p", config.Port)
	}
	for _, id := range config.IdentityFiles {
		args = append(args, "-i", id)
	}
	if config.ProxyCommand != "" {
		args = append(args, "-o", "ProxyCommand="+config.ProxyCommand)
	}
	if config.ProxyJump != "" {
		args = append(args, "-J", config.ProxyJump)
	}
	args = append(args, "-o", "NumberOfPasswordPrompts=1")
	return args
}

// FindPublicKey returns the path to the public key for the given config.
// It checks IdentityFiles first (appending .pub), then falls back to ~/.ssh/id_*.pub.
func FindPublicKey(config *HostConfig) (string, error) {
	for _, id := range config.IdentityFiles {
		if candidate := id + ".pub"; fileExists(candidate) {
			return candidate, nil
		}
	}
	k, err := FindDefaultKey()
	pk := k + ".pub"
	if err != nil || !fileExists(pk) {
		return "", fmt.Errorf("no public key found")
	}

	return pk, nil
}

func FindDefaultKey() (string, error) {
	home, _ := os.UserHomeDir()
	for _, name := range []string{"id_ed25519", "id_rsa", "id_ecdsa", "id_dsa"} {
		if p := filepath.Join(home, ".ssh", name); fileExists(p) {
			return p, nil
		}
	}

	return "", fmt.Errorf("no key found")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// validHostRe allows hostnames, FQDNs, plain IPv4 and IPv6 addresses
// (brackets and colons are kept after net.SplitHostPort strips the brackets).
var validHostRe = regexp.MustCompile(`^[a-zA-Z0-9._:\-\[\]%]+$`)

// validUserRe follows POSIX portable filename character set for usernames.
var validUserRe = regexp.MustCompile(`^[a-zA-Z0-9._\-]+$`)

// ParseConnection parses a connection string like "user:pass@host:port"
// and returns a temporary HostConfig if valid.
// Supported formats:
//
//	[user[:password]@]host[:port]
func ParseConnection(s string) *HostConfig {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var user, pass, host, port string

	// Split user info from host info
	if i := strings.LastIndex(s, "@"); i != -1 {
		userInfo := s[:i]
		hostInfo := s[i+1:]

		if j := strings.Index(userInfo, ":"); j != -1 {
			user = userInfo[:j]
			pass = userInfo[j+1:]
		} else {
			user = userInfo
		}
		s = hostInfo
	}

	// Split host from port
	if h, p, err := net.SplitHostPort(s); err == nil {
		host = h
		port = p
	} else {
		host = s
	}

	if host == "" {
		return nil
	}

	// Validate host contains only valid characters
	if !validHostRe.MatchString(host) {
		return nil
	}

	// Validate user contains only valid characters (if provided)
	if user != "" && !validUserRe.MatchString(user) {
		return nil
	}

	if port == "" {
		port = "22"
	}

	// Validate port is a number in the valid range
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
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
