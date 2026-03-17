package system

import (
	"os"
	"runtime"
	"strings"
)

const IsWindows = runtime.GOOS == "windows"
const IsLinux = runtime.GOOS == "linux"
const IsDarwin = runtime.GOOS == "darwin"

var IsWSL = detectWSL()

func detectWSL() bool {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}
