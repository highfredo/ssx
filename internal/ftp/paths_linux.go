package ftp

var filezillaPaths = []string{
	"filezilla",
	"/usr/bin/filezilla",
	"/usr/local/bin/filezilla",
	// Windows (via WSL)
	"/mnt/c/Program Files/FileZilla FTP Client/filezilla.exe",
	"/mnt/c/Program Files (x86)/FileZilla FTP Client/filezilla.exe",
}

var winscpPaths = []string{
	// Windows (via WSL)
	"/mnt/c/Program Files (x86)/WinSCP/WinSCP.exe",
	"/mnt/c/Program Files/WinSCP/WinSCP.exe",
	"/mnt/c/Program Files (x86)/WinSCP/winscp.exe",
	"/mnt/c/Program Files/WinSCP/winscp.exe",
}
