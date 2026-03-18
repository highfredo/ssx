package hosts

import (
	"strings"

	"github.com/highfredo/ssx/internal/ssh"
)

type item struct {
	host           *ssh.HostConfig
	isQuickConnect bool
}

func (i item) Title() string {
	return i.host.Title
}

func (i item) Description() string {
	description := make([]string, 0)

	if i.host.User != "" {
		description = append(description, i.host.User, "@")
	}
	description = append(description, i.host.Hostname)
	if i.host.Port != "22" {
		description = append(description, ":", i.host.Port)
	}

	return strings.Join(description, "")
}

func (i item) FilterValue() string {
	if i.isQuickConnect {
		return quickConnectFilterValue
	}
	parts := []string{i.host.Name, i.host.Hostname, i.host.User, i.host.Port}
	for _, t := range i.host.Tags {
		parts = append(parts, t.Name)
	}
	return strings.Join(parts, " ")
}

const quickConnectFilterValue = "__QUICK_HOST__"
