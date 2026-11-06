package hosts

import (
	"strings"

	"github.com/highfredo/ssx/internal/ssh"
)

type item struct {
	host *ssh.HostConfig
}

func (i item) Title() string {
	return i.host.Title
}

// Description renders user@hostname:port, omitting empty user and the
// default port.
func (i item) Description() string {
	var sb strings.Builder
	if i.host.User != "" {
		sb.WriteString(i.host.User)
		sb.WriteByte('@')
	}
	sb.WriteString(i.host.Hostname)
	if i.host.Port != ssh.DefaultPort {
		sb.WriteByte(':')
		sb.WriteString(i.host.Port)
	}
	return sb.String()
}

func (i item) FilterValue() string {
	parts := []string{i.host.Title}
	for _, t := range i.host.Tags {
		parts = append(parts, t.Name)
	}
	parts = append(parts, i.host.Name, i.host.Hostname, i.host.User, i.host.Port)
	return strings.Join(parts, " ")
}
