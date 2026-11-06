package system

import (
	"os"
	"runtime"
	"strings"
)

// Platform flags evaluated once at startup.
const IsWindows = runtime.GOOS == "windows"
const IsLinux = runtime.GOOS == "linux"
const IsDarwin = runtime.GOOS == "darwin"

// IsWSL reports whether the Linux binary runs under Windows Subsystem for Linux.
var IsWSL = detectWSL()

func detectWSL() bool {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}
