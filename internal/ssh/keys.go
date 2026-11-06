package ssh

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/highfredo/ssx/internal/paths"
	gossh "golang.org/x/crypto/ssh"
)

// CopyID builds an ssh command that installs the local public key on the
// remote host in a single connection, avoiding the ssh-copy-id double-connect
// hang. It reads the public key locally and embeds it in the remote command.
func CopyID(config *HostConfig) (*exec.Cmd, error) {
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
	if !config.HasPassword() {
		// Never block on an interactive prompt: surface ErrAuthFailed instead.
		sshArgs = append(sshArgs, "-o", "BatchMode=yes")
	}

	return PrepareCmdWithCommand(config, sshArgs, []string{remoteCmd}), nil
}

// RunCopyID installs the public key on the host without touching the
// terminal. It returns ErrAuthFailed when ssh rejects the credentials.
func RunCopyID(config *HostConfig) error {
	cmd, err := CopyID(config)
	if err != nil {
		return err
	}
	stderr := newLineWriter(func(line string) { slog.Debug("ssh copy-id", "stderr", line) })
	cmd.Stderr = stderr
	cmd.Stdin = nil
	if err := cmd.Run(); err != nil {
		if err = classifyExit(err, stderr.String()); errors.Is(err, ErrAuthFailed) {
			return err
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
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
	if err != nil {
		return "", fmt.Errorf("no public key found: %w", err)
	}
	if pk := k + ".pub"; fileExists(pk) {
		return pk, nil
	}
	return "", fmt.Errorf("no public key found for %s", k)
}

// FindDefaultKey returns the path to the first existing default private key in ~/.ssh.
func FindDefaultKey() (string, error) {
	for _, name := range []string{"id_ed25519", "id_rsa", "id_ecdsa", "id_dsa"} {
		if p := filepath.Join(paths.SSHDir(), name); fileExists(p) {
			return p, nil
		}
	}

	return "", fmt.Errorf("no key found")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// OpenSSHKeyToPuttyKey converts OpenSSH private key PEM data into unencrypted PuTTY PPK v2 format.
// If comment is empty, "imported-openssh-key" is used.
func OpenSSHKeyToPuttyKey(pemData []byte, comment string) ([]byte, error) {
	rawKey, err := gossh.ParseRawPrivateKey(pemData)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	signer, err := gossh.NewSignerFromKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("create signer: %w", err)
	}

	algo := signer.PublicKey().Type()
	pubBlob := signer.PublicKey().Marshal()

	var privBlob []byte
	switch k := rawKey.(type) {
	case *ed25519.PrivateKey:
		// blob = string(seed||pubkey) — 64 bytes
		privBlob = wireString(*k)
	case *rsa.PrivateKey:
		// blob = mpint(d) + mpint(p) + mpint(q) + mpint(iqmp)
		// PuTTY expects p > q
		p, q := k.Primes[0], k.Primes[1]
		if p.Cmp(q) < 0 {
			p, q = q, p
		}
		iqmp := new(big.Int).ModInverse(q, p)
		privBlob = append(privBlob, mpint(k.D)...)
		privBlob = append(privBlob, mpint(p)...)
		privBlob = append(privBlob, mpint(q)...)
		privBlob = append(privBlob, mpint(iqmp)...)
	case *ecdsa.PrivateKey:
		// blob = mpint(d) — ECDH() avoids deprecated D field
		ecdhKey, err := k.ECDH()
		if err != nil {
			return nil, fmt.Errorf("derive ECDH key: %w", err)
		}
		privBlob = mpint(new(big.Int).SetBytes(ecdhKey.Bytes()))
	default:
		return nil, fmt.Errorf("unsupported key type: %T", rawKey)
	}

	if comment == "" {
		comment = "imported-openssh-key"
	}

	return buildPPK(algo, comment, pubBlob, privBlob), nil
}

// buildPPK constructs the unencrypted PPK v2 file contents.
func buildPPK(algo, comment string, pubBlob, privBlob []byte) []byte {
	const encryption = "none"

	mac := macSHA1(algo, encryption, comment, pubBlob, privBlob, "")
	pubLines := b64Lines(pubBlob)
	privLines := b64Lines(privBlob)

	var sb strings.Builder
	sb.WriteString("PuTTY-User-Key-File-2: " + algo + "\n")
	sb.WriteString("Encryption: " + encryption + "\n")
	sb.WriteString("Comment: " + comment + "\n")
	sb.WriteString(fmt.Sprintf("Public-Lines: %d\n", len(pubLines)))
	for _, l := range pubLines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString(fmt.Sprintf("Private-Lines: %d\n", len(privLines)))
	for _, l := range privLines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString("Private-MAC: " + mac + "\n")
	return []byte(sb.String())
}

// macSHA1 computes the PPK v2 integrity HMAC-SHA1.
func macSHA1(algo, encryption, comment string, pubBlob, privBlob []byte, passphrase string) string {
	h := sha1.New()
	h.Write([]byte("putty-private-key-file-mac-key"))
	h.Write([]byte(passphrase))
	macKey := h.Sum(nil)

	var data []byte
	data = append(data, wireString([]byte(algo))...)
	data = append(data, wireString([]byte(encryption))...)
	data = append(data, wireString([]byte(comment))...)
	data = append(data, wireString(pubBlob)...)
	data = append(data, wireString(privBlob)...)

	mac := hmac.New(sha1.New, macKey)
	mac.Write(data)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// wireString serializes []byte as an SSH wire string: uint32(len) + data.
func wireString(data []byte) []byte {
	b := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(b, uint32(len(data)))
	copy(b[4:], data)
	return b
}

// mpint serializes a *big.Int as an SSH mpint: uint32(len) + bytes with leading sign byte if needed.
func mpint(n *big.Int) []byte {
	raw := n.Bytes()
	if len(raw) > 0 && raw[0]&0x80 != 0 {
		raw = append([]byte{0x00}, raw...)
	}
	return wireString(raw)
}

// b64Lines encodes data in base64 split into 64-character lines.
func b64Lines(data []byte) []string {
	enc := base64.StdEncoding.EncodeToString(data)
	var lines []string
	for len(enc) > 64 {
		lines = append(lines, enc[:64])
		enc = enc[64:]
	}
	if len(enc) > 0 {
		lines = append(lines, enc)
	}
	return lines
}
