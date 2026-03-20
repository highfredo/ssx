package ssh

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type PortState int

const (
	PortClosed PortState = iota
	PortOpened
)

type PortStatus struct {
	Number string
	State  PortState
	PID    int
	Owner  string
}

func GetPortStatus(port string) (portStatus PortStatus, err error) {
	pid, owner, err := localPortOwner(port)
	state := PortClosed
	if pid > -1 {
		state = PortOpened
	}
	return PortStatus{
		Number: port,
		PID:    pid,
		Owner:  owner,
		State:  state,
	}, err
}

func KillPortOwner(port string) error {
	pid, owner, err := localPortOwner(port)
	if err != nil {
		return fmt.Errorf("check port Owner: %w", err)
	}
	if owner == "" {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := proc.Kill(); err != nil {
		return fmt.Errorf("kill process %d: %w", pid, err)
	}
	slog.Info("killed port Owner", "port", port, "PID", pid, "command", owner)
	return nil
}

/*
**************

	Utils

**************
*/

// localPortOwner dispatches to the appropriate OS-specific implementation.
func localPortOwner(port string) (pid int, owner string, err error) {
	switch runtime.GOOS {
	case "windows":
		return localPortOwnerByNetstat(port)
	case "darwin":
		return localPortOwnerByLSOF(port)
	default:
		return localPortOwnerBySS(port)
	}
}

var ssUsersPattern = regexp.MustCompile(`users:\(\("([^"]+)",(?i:pid)=(\d+)`)

func localPortOwnerBySS(port string) (pid int, owner string, err error) {
	cmd := exec.Command("ss", "-ltnpH", fmt.Sprintf("sport = :%s", port))
	out, err := cmd.Output()
	if err != nil {
		return -1, "", err
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return -1, "", nil
	}

	match := ssUsersPattern.FindStringSubmatch(line)
	if len(match) < 3 {
		return -1, "", nil
	}
	spid, serr := strconv.Atoi(match[2])
	return spid, match[1], serr
}

func localPortOwnerByNetstat(port string) (pid int, owner string, err error) {
	cmd := exec.Command("netstat", "-ano")
	out, err := cmd.Output()
	if err != nil {
		return -1, "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		// netstat -ano format: Proto  LocalAddr  ForeignAddr  State  PID
		if len(fields) < 5 {
			continue
		}
		if !strings.EqualFold(fields[3], "LISTENING") {
			continue
		}
		if !strings.HasSuffix(fields[1], ":"+port) {
			continue
		}
		pidNum, err := strconv.Atoi(fields[4])
		if err != nil {
			return -1, "", err
		}
		name, err := processNameByPID(pidNum)
		return pidNum, name, err
	}
	return -1, "", nil
}

func processNameByPID(pid int) (string, error) {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return "", nil
	}
	parts := strings.SplitN(line, ",", 2)
	return strings.Trim(parts[0], `"`), nil
}

func localPortOwnerByLSOF(port string) (pid int, owner string, err error) {
	cmd := exec.Command("lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN")
	out, err := cmd.Output()
	if err != nil {
		// lsof exits with 1 when there are no matches.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return -1, "", nil
		}
		return -1, "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// First line is the header; skip it.
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pidNum, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		return pidNum, fields[0], nil
	}
	return -1, "", nil
}
