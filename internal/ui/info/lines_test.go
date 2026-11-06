package info

import (
	"testing"

	"github.com/highfredo/ssx/internal/ssh"
)

func TestGroupLines(t *testing.T) {
	lines := []parsedLine{
		{key: "hostname", value: "example.com"},
		{key: "port", value: "2222"},
		{key: "proxyjump", value: "bastion"},
		{key: "someunknownkey", value: "x"},
	}

	tests := []struct {
		name   string
		query  string
		titles []string
		counts []int
	}{
		{"no filter", "", []string{"🔌 Connection", "🔀 Proxy", "⚙️ Other"}, []int{2, 1, 1}},
		{"filter by value", "bastion", []string{"🔀 Proxy"}, []int{1}},
		{"filter by key", "port", []string{"🔌 Connection"}, []int{1}},
		{"no match", "zzz", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := groupLines(lines, tc.query)
			if len(got) != len(tc.titles) {
				t.Fatalf("got %d sections, want %d: %+v", len(got), len(tc.titles), got)
			}
			for i, sec := range got {
				if sec.title != tc.titles[i] {
					t.Errorf("section %d title = %q, want %q", i, sec.title, tc.titles[i])
				}
				if len(sec.lines) != tc.counts[i] {
					t.Errorf("section %q has %d lines, want %d", sec.title, len(sec.lines), tc.counts[i])
				}
			}
		})
	}
}

func TestBuildMagicLines(t *testing.T) {
	host := &ssh.HostConfig{
		Tags:            []ssh.Tag{{Name: "dev"}, {Name: "aws", Color: "#ff0"}},
		Password:        "secret",
		PasswordCommand: "pass show x",
	}
	got := buildMagicLines(host)
	want := []parsedLine{
		{key: "tags", value: "dev, aws:#ff0"},
		{key: "password", value: "***"},
		{key: "passwordCommand", value: "pass show x"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %+v", len(got), len(want), got)
	}
	if len(got[0].tags) != 2 {
		t.Errorf("tags line should carry the host tags, got %+v", got[0].tags)
	}
	for i := range want {
		if got[i].key != want[i].key || got[i].value != want[i].value {
			t.Errorf("line %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
