// Package ppk convierte claves privadas OpenSSH al formato PPK v2 de PuTTY.
package ppk

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	gossh "golang.org/x/crypto/ssh"
)

// FromOpenSSH convierte datos PEM de clave privada OpenSSH al formato PPK v2 sin cifrado.
// comment es el comentario que se incrustará en el archivo; si está vacío se usa "imported-openssh-key".
func FromOpenSSH(pemData []byte, comment string) ([]byte, error) {
	rawKey, err := gossh.ParseRawPrivateKey(pemData)
	if err != nil {
		return nil, fmt.Errorf("error al parsear la clave privada: %w", err)
	}

	signer, err := gossh.NewSignerFromKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("error al crear signer: %w", err)
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
		// PuTTY espera p > q
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
		// blob = mpint(d) — ECDH() evita el campo D deprecado
		ecdhKey, err := k.ECDH()
		if err != nil {
			return nil, fmt.Errorf("error al obtener clave ECDH: %w", err)
		}
		privBlob = mpint(new(big.Int).SetBytes(ecdhKey.Bytes()))
	default:
		return nil, fmt.Errorf("tipo de clave no soportado: %T", rawKey)
	}

	if comment == "" {
		comment = "imported-openssh-key"
	}

	return build(algo, comment, pubBlob, privBlob), nil
}

// build construye el contenido del archivo PPK v2 sin cifrado.
func build(algo, comment string, pubBlob, privBlob []byte) []byte {
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

// macSHA1 calcula el HMAC-SHA1 de integridad del archivo PPK v2.
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

// wireString serializa []byte como SSH wire string: uint32(len) + data.
func wireString(data []byte) []byte {
	b := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(b, uint32(len(data)))
	copy(b[4:], data)
	return b
}

// mpint serializa un *big.Int como SSH mpint: uint32(len) + bytes con bit de signo si es necesario.
func mpint(n *big.Int) []byte {
	raw := n.Bytes()
	if len(raw) > 0 && raw[0]&0x80 != 0 {
		raw = append([]byte{0x00}, raw...)
	}
	return wireString(raw)
}

// b64Lines codifica data en base64 partiendo en líneas de 64 caracteres.
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
