package ftp

import (
	"os"
	"path/filepath"

	"github.com/highfredo/ssx/internal/paths"
)

var filezillaPaths = []string{
	"filezilla.exe",
	`C:\Program Files\FileZilla FTP Client\filezilla.exe`,
	`C:\Program Files (x86)\FileZilla FTP Client\filezilla.exe`,
}

var winscpPaths = []string{
	"WinSCP.exe",
	"winscp.exe",
	filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "WinSCP", "WinSCP.exe"),
	filepath.Join(paths.Home(), "AppData", "Local", "Programs", "WinSCP", "WinSCP.exe"),
	`C:\Program Files (x86)\WinSCP\WinSCP.exe`,
	`C:\Program Files\WinSCP\WinSCP.exe`,
}
