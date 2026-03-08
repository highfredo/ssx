package ssh

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
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
	pid, owner, err := localPortOwnerBySS(port)
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
	pid, owner, err := localPortOwnerBySS(port)
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
	return parseSSOwnerLine(line)
}

var ssUsersPattern = regexp.MustCompile(`users:\(\("([^"]+)",(?i:pid)=(\d+)`)

func parseSSOwnerLine(line string) (pid int, owner string, err error) {
	match := ssUsersPattern.FindStringSubmatch(line)
	if len(match) < 3 {
		return -1, "", nil
	}
	spid, serr := strconv.Atoi(match[2])
	return spid, match[1], serr
}
