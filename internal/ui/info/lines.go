package info

import (
	"os/exec"
	"strings"

	"github.com/highfredo/ssx/internal/ssh"
)

// parsedLine holds a single ssh -G line split into key and value.
type parsedLine struct {
	key   string
	value string
	tags  []ssh.Tag // when set, value is rendered as tag chips
}

// section is a titled group of lines ready to be rendered.
type section struct {
	title string
	lines []parsedLine
}

// groupLines distributes lines into sshSections (plus a trailing "Other"
// section for unassigned keys), keeping only those matching the lowercase
// query q. Sections without matching lines are omitted.
func groupLines(lines []parsedLine, q string) []section {
	byKey := make(map[string][]parsedLine, len(lines))
	for _, l := range lines {
		byKey[l.key] = append(byKey[l.key], l)
	}

	assigned := make(map[string]bool)
	var out []section
	for _, def := range sshSections {
		var matched []parsedLine
		for _, k := range def.keys {
			assigned[k] = true
			for _, l := range byKey[k] {
				if l.matches(q) {
					matched = append(matched, l)
				}
			}
		}
		if len(matched) > 0 {
			out = append(out, section{title: def.title, lines: matched})
		}
	}

	var other []parsedLine
	for _, l := range lines {
		if !assigned[l.key] && l.matches(q) {
			other = append(other, l)
		}
	}
	if len(other) > 0 {
		out = append(out, section{title: "⚙️ Other", lines: other})
	}
	return out
}

// matches reports whether the lowercase query q occurs in the key or value.
// An empty query matches everything.
func (l parsedLine) matches(q string) bool {
	return q == "" ||
		strings.Contains(strings.ToLower(l.key), q) ||
		strings.Contains(strings.ToLower(l.value), q)
}

// parseSSHG runs «ssh -G <name>» and returns the parsed lines.
func parseSSHG(host *ssh.HostConfig) []parsedLine {
	out, err := exec.Command("ssh", "-G", host.Name).Output()
	if err != nil {
		return []parsedLine{{key: "error", value: err.Error()}}
	}
	var lines []parsedLine
	for raw := range strings.SplitSeq(strings.TrimRight(string(out), "\n"), "\n") {
		parts := strings.SplitN(raw, " ", 2)
		if len(parts) == 2 {
			lines = append(lines, parsedLine{key: parts[0], value: parts[1]})
		} else if raw != "" {
			lines = append(lines, parsedLine{key: raw})
		}
	}
	return lines
}

// buildMagicLines converts HostConfig magic-comment fields into parsedLines
// so they appear in the "🪄 ssx" section and are searchable.
func buildMagicLines(host *ssh.HostConfig) []parsedLine {
	var lines []parsedLine

	if len(host.Tags) > 0 {
		var parts []string
		for _, t := range host.Tags {
			if t.Color != "" {
				parts = append(parts, t.Name+":"+t.Color)
			} else {
				parts = append(parts, t.Name)
			}
		}
		lines = append(lines, parsedLine{key: "tags", value: strings.Join(parts, ", "), tags: host.Tags})
	}

	if host.Password != "" {
		lines = append(lines, parsedLine{key: "password", value: "***"})
	}

	if host.PasswordCommand != "" {
		lines = append(lines, parsedLine{key: "passwordCommand", value: host.PasswordCommand})
	}

	return lines
}
