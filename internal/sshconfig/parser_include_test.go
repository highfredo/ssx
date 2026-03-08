package sshconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigIncludeLoadsHostsAndAnnotations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(filepath.Join(sshDir, "conf.d"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeTestFile(t, filepath.Join(sshDir, "config"), `
Host local
    HostName 127.0.0.1

Include conf.d/*.conf
`)
	writeTestFile(t, filepath.Join(sshDir, "conf.d", "app.conf"), `
Host mini
    #>tags: included, personal
    HostName 192.168.10.2
    LocalForward 8080 localhost:8080 # web server
`)

	hosts, err := ParseConfig()
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	byName := make(map[string]*Host, len(hosts))
	for _, host := range hosts {
		byName[host.Name] = host
	}

	if _, ok := byName["local"]; !ok {
		t.Fatalf("expected host local in %+v", byName)
	}
	mini, ok := byName["mini"]
	if !ok {
		t.Fatalf("expected host mini in %+v", byName)
	}

	assertTags(t, mini.Tags, []string{"included", "personal"})
	if len(mini.Tunnels) != 1 {
		t.Fatalf("expected one tunnel for mini, got %v", mini.Tunnels)
	}
	if mini.Tunnels[0].Description != "web server" {
		t.Fatalf("unexpected tunnel description: got %q", mini.Tunnels[0].Description)
	}
}

func TestParseConfigIncludeRecursiveError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeTestFile(t, filepath.Join(sshDir, "config"), `
Include loop.conf
`)
	writeTestFile(t, filepath.Join(sshDir, "loop.conf"), `
Include config
`)

	_, err := ParseConfig()
	if err == nil {
		t.Fatalf("expected recursive include error")
	}
	if !strings.Contains(err.Error(), "recursive include") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
