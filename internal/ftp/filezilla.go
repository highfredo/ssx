package ftp

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/highfredo/ssx/internal/ssh"
)

// filezillaPaths lista los ejecutables de FileZilla por orden de preferencia.
// Incluye rutas Windows (WSL), Linux y macOS.
var filezillaPaths = []string{
	// Windows (via WSL)
	"/mnt/c/Program Files/FileZilla FTP Client/filezilla.exe",
	"/mnt/c/Program Files (x86)/FileZilla FTP Client/filezilla.exe",
	// Linux
	"/usr/bin/filezilla",
	"/usr/local/bin/filezilla",
	// macOS
	"/Applications/FileZilla.app/Contents/MacOS/filezilla",
}

// isWindowsExe informa si el ejecutable es un binario Windows
// (detectado por extensión .exe o por estar bajo /mnt/).
func isWindowsExe(exe string) bool {
	lower := strings.ToLower(exe)
	return strings.HasSuffix(lower, ".exe") || strings.HasPrefix(exe, "/mnt/")
}

// launchFilezilla crea un directorio de configuración temporal con un
// sitemanager.xml que incluye host, credenciales y clave, luego lanza:
//
//	filezilla [--config-dir <dir>] --site=temp
//
// En entornos Windows (WSL) las rutas se convierten con wslpath y la clave
// se convierte a PPK. En Linux/macOS se usan rutas nativas y claves OpenSSH.
// El directorio temporal se elimina cuando FileZilla cierra.
func launchFilezilla(h *ssh.HostConfig, exe string) error {
	if isWindowsExe(exe) {
		return launchFilezillaWindows(h, exe)
	}
	return launchFilezillaUnix(h, exe)
}

// launchFilezillaWindows abre FileZilla en Windows/WSL usando solo la URL SFTP.
// Windows FileZilla no soporta --datadir de forma fiable; la autenticación por
// clave debe estar configurada en el site manager de FileZilla o en Pageant.
func launchFilezillaWindows(h *ssh.HostConfig, exe string) error {
	slog.Info("launching FileZilla (Windows)", "host", h.Hostname)
	url := buildURL(h.User, h.Hostname, h.Port, resolvePassword(h))
	cmd := exec.Command(exe, url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("filezilla: arranque: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// launchFilezillaUnix abre FileZilla en Linux/macOS creando un directorio de
// configuración temporal con sitemanager.xml. Usa --datadir (flag oficial de
// FileZilla) para que lea la config desde ese directorio.
func launchFilezillaUnix(h *ssh.HostConfig, exe string) error {
	tempDir, err := os.MkdirTemp("", "ssx-filezilla-*")
	if err != nil {
		return fmt.Errorf("filezilla: crear directorio temporal: %w", err)
	}

	xmlData, err := buildSiteManagerXML(h.User, h.Hostname, h.Port, resolvePassword(h), resolveKeyPath(h))
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return fmt.Errorf("filezilla: sitemanager.xml: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "sitemanager.xml"), xmlData, 0o600); err != nil {
		_ = os.RemoveAll(tempDir)
		return fmt.Errorf("filezilla: escribir sitemanager.xml: %w", err)
	}

	cmd := exec.Command(exe, "--datadir="+tempDir, "--site=temp")
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(tempDir)
		return fmt.Errorf("filezilla: arranque: %w", err)
	}
	go func() {
		_ = cmd.Wait()
		_ = os.RemoveAll(tempDir)
	}()

	slog.Info("launching FileZilla (Unix)", "host", h.Hostname)
	return nil
}

// buildSiteManagerXML genera el sitemanager.xml de FileZilla para un sitio
// llamado "temp". Logontype:
//   - 5 (Key file)    si se proporciona keyPath
//   - 1 (Normal)      si solo hay contraseña
//   - 4 (Interactive) si no hay ninguno
func buildSiteManagerXML(user, host, port, password, keyPath string) ([]byte, error) {
	type passElem struct {
		Encoding string `xml:"encoding,attr"`
		Value    string `xml:",chardata"`
	}
	type server struct {
		XMLName      xml.Name  `xml:"Server"`
		Host         string    `xml:"Host"`
		Port         string    `xml:"Port"`
		Protocol     int       `xml:"Protocol"` // 1 = SFTP
		Type         int       `xml:"Type"`
		User         string    `xml:"User"`
		Pass         *passElem `xml:"Pass,omitempty"`
		Keyfile      string    `xml:"Keyfile,omitempty"`
		Logontype    int       `xml:"Logontype"`
		Name         string    `xml:"Name"`
		SyncBrowsing int       `xml:"SyncBrowsing"`
	}
	type root struct {
		XMLName xml.Name `xml:"FileZilla3"`
		Servers struct {
			XMLName xml.Name `xml:"Servers"`
			Server  server
		}
	}

	var logontype int
	var p *passElem
	switch {
	case keyPath != "":
		logontype = 5
	case password != "":
		logontype = 1
		p = &passElem{
			Encoding: "base64",
			Value:    base64.StdEncoding.EncodeToString([]byte(password)),
		}
	default:
		logontype = 4
	}

	doc := root{}
	doc.Servers.Server = server{
		Host:      host,
		Port:      port,
		Protocol:  1,
		User:      user,
		Pass:      p,
		Keyfile:   keyPath,
		Logontype: logontype,
		Name:      "temp",
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}
