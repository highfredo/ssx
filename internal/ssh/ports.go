package ssh

import (
	"fmt"
	"strconv"
	"strings"

	gopsnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// portOwner describes a process holding a listening TCP port.
type portOwner struct {
	PID     int32 // 0 when the owner is not visible to the current user (other user / root).
	Name    string
	Cmdline string
}

func (o portOwner) isSSH() bool {
	return isSSHProcess(o.Name)
}

// portSnapshot is a point-in-time view of the local listening TCP ports.
// Process details are resolved lazily and cached per PID so that callers can
// query several ports from a single gopsutil scan.
type portSnapshot struct {
	conns  []gopsnet.ConnectionStat
	owners map[int32]*portOwner // nil entry: process vanished or is not inspectable
	ssh    map[string]string    // listening port → cmdline, for ssh processes only
}

func snapshotPorts() (*portSnapshot, error) {
	conns, err := gopsnet.Connections("tcp")
	if err != nil {
		return nil, fmt.Errorf("list tcp connections: %w", err)
	}
	return newPortSnapshot(conns), nil
}

func newPortSnapshot(conns []gopsnet.ConnectionStat) *portSnapshot {
	listening := conns[:0:0]
	for _, c := range conns {
		if strings.EqualFold(c.Status, "LISTEN") {
			listening = append(listening, c)
		}
	}
	return &portSnapshot{conns: listening, owners: make(map[int32]*portOwner)}
}

// ownersOf returns one entry per listening socket bound to port, in scan
// order. Sockets whose process can no longer be inspected are omitted.
func (s *portSnapshot) ownersOf(port uint32) []portOwner {
	var owners []portOwner
	for _, c := range s.conns {
		if c.Laddr.Port != port {
			continue
		}
		if o := s.owner(c.Pid); o != nil {
			owners = append(owners, *o)
		}
	}
	return owners
}

// isFree reports whether no listening socket is bound to port.
func (s *portSnapshot) isFree(port uint32) bool {
	for _, c := range s.conns {
		if c.Laddr.Port == port {
			return false
		}
	}
	return true
}

// owner resolves the process behind pid, caching the result.
func (s *portSnapshot) owner(pid int32) *portOwner {
	if pid <= 0 {
		return &portOwner{}
	}
	if o, ok := s.owners[pid]; ok {
		return o
	}
	var o *portOwner
	if proc, err := process.NewProcess(pid); err == nil {
		name, _ := proc.Name()
		o = &portOwner{PID: pid, Name: name}
		if o.isSSH() {
			o.Cmdline, _ = proc.Cmdline()
		}
	}
	s.owners[pid] = o
	return o
}

// sshCmdlines maps every listening port held by an ssh process to the command
// line of that process. The first socket seen per port wins.
func (s *portSnapshot) sshCmdlines() map[string]string {
	if s.ssh != nil {
		return s.ssh
	}
	s.ssh = make(map[string]string)
	for _, c := range s.conns {
		port := strconv.FormatUint(uint64(c.Laddr.Port), 10)
		if _, seen := s.ssh[port]; seen || c.Pid <= 0 {
			continue
		}
		if o := s.owner(c.Pid); o != nil && o.isSSH() {
			s.ssh[port] = o.Cmdline
		}
	}
	return s.ssh
}

func parsePort(s string) (uint32, error) {
	p, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: %w", s, err)
	}
	return uint32(p), nil
}

func isSSHProcess(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == "ssh" || name == "ssh.exe"
}

// cmdlineMatchesHost reports whether an ssh command line targets host, either
// by alias or by hostname (optionally prefixed with "user@").
// A nil host matches any command line.
func cmdlineMatchesHost(cmdline string, host *HostConfig) bool {
	if host == nil {
		return true
	}
	for f := range strings.FieldsSeq(cmdline) {
		f = strings.Trim(f, `"'`)
		if _, after, ok := strings.Cut(f, "@"); ok {
			f = after
		}
		if f == host.Name || f == host.Hostname {
			return true
		}
	}
	return false
}
