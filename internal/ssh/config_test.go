package ssh

import (
	"strings"
	"testing"

	gossh "github.com/kevinburke/ssh_config"
)

func decodeTestConfig(t *testing.T, lines ...string) *gossh.Config {
	t.Helper()
	cfg, err := gossh.Decode(strings.NewReader(strings.Join(lines, "\n")))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	return cfg
}

func TestCollectHostsConfigLine(t *testing.T) {
	cfg := decodeTestConfig(t,
		"# Global comment",
		"",
		"Host alpha",
		"  HostName 10.0.0.1",
		"",
		"Host beta",
		"  HostName 10.0.0.2",
	)

	hosts := newHostCollector(cfg).collect(cfg, "/tmp/test_ssh_config")
	if len(hosts) != 2 {
		t.Fatalf("got %d hosts, want 2", len(hosts))
	}
	if hosts[0].Name != "alpha" || hosts[0].ConfigFile != "/tmp/test_ssh_config" || hosts[0].ConfigLine != 3 {
		t.Errorf("alpha = %+v, want line 3 in /tmp/test_ssh_config", hosts[0])
	}
	if hosts[1].Name != "beta" || hosts[1].ConfigFile != "/tmp/test_ssh_config" || hosts[1].ConfigLine != 6 {
		t.Errorf("beta = %+v, want line 6 in /tmp/test_ssh_config", hosts[1])
	}
}

func TestCollectHostsMagicCommentsAndTunnels(t *testing.T) {
	cfg := decodeTestConfig(t,
		"Host *",
		"  #>tags:shared",
		"  #>password:global-secret",
		"",
		"Host web api # Web tier",
		"  HostName 10.0.0.1",
		"  Port 2222",
		"  #>tags:prod:red,shared:blue",
		"  LocalForward 8080 localhost:80 # http",
		"  DynamicForward 1080",
		"",
		"Host secret",
		"  #>hidden:true",
		"",
		"Host wild-*",
		"  User nobody",
	)

	hosts := Hosts(newHostCollector(cfg).collect(cfg, "/tmp/cfg"))
	if got := len(hosts); got != 2 {
		t.Fatalf("got %d hosts, want 2 (hidden and wildcard skipped): %+v", got, hosts)
	}

	web := hosts.Get("web")
	if web == nil {
		t.Fatal("host web not found")
	}
	if web.Title != "Web tier (web)" {
		t.Errorf("Title = %q, want %q", web.Title, "Web tier (web)")
	}
	if web.Hostname != "10.0.0.1" || web.Port != "2222" {
		t.Errorf("Hostname:Port = %s:%s, want 10.0.0.1:2222", web.Hostname, web.Port)
	}
	if web.Password != "global-secret" {
		t.Errorf("Password = %q, want inherited global-secret", web.Password)
	}
	if web.IsAdHoc() {
		t.Error("IsAdHoc() = true for a configured host")
	}

	wantTags := []Tag{{Name: "prod", Color: "red"}, {Name: "shared", Color: "blue"}}
	if len(web.Tags) != len(wantTags) {
		t.Fatalf("Tags = %+v, want %+v", web.Tags, wantTags)
	}
	for i, want := range wantTags {
		if web.Tags[i] != want {
			t.Errorf("Tags[%d] = %+v, want %+v (host color must win over wildcard)", i, web.Tags[i], want)
		}
	}

	if len(web.Tunnels) != 2 {
		t.Fatalf("Tunnels = %+v, want 2", web.Tunnels)
	}
	local := web.Tunnels[0]
	if local.Type != TunnelTypeLocal || local.Host != "web" || local.LocalPort != "8080" ||
		local.RemoteHost != "localhost" || local.RemotePort != "80" || local.Description != "http" {
		t.Errorf("LocalForward tunnel = %+v", local)
	}
	if dyn := web.Tunnels[1]; dyn.Type != TunnelTypeDynamic || dyn.LocalPort != "1080" {
		t.Errorf("DynamicForward tunnel = %+v", dyn)
	}

	if hosts.Get("secret") != nil {
		t.Error("hidden host must not be listed")
	}
	if hosts.Get("missing") != nil {
		t.Error("Get(missing) must return nil")
	}
}

func TestParseForward(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    Tunnel
		wantOK  bool
		tunType TunnelType
	}{
		{
			name:    "plain",
			spec:    "8080 localhost:80",
			tunType: TunnelTypeLocal,
			want:    Tunnel{Type: TunnelTypeLocal, LocalPort: "8080", RemoteHost: "localhost", RemotePort: "80"},
			wantOK:  true,
		},
		{
			name:    "bind address stripped",
			spec:    "127.0.0.1:8080 db.internal:5432",
			tunType: TunnelTypeRemote,
			want:    Tunnel{Type: TunnelTypeRemote, LocalPort: "8080", RemoteHost: "db.internal", RemotePort: "5432"},
			wantOK:  true,
		},
		{
			name:    "ipv6 remote",
			spec:    "9000 [::1]:9001",
			tunType: TunnelTypeLocal,
			want:    Tunnel{Type: TunnelTypeLocal, LocalPort: "9000", RemoteHost: "::1", RemotePort: "9001"},
			wantOK:  true,
		},
		{name: "missing remote", spec: "8080", tunType: TunnelTypeLocal},
		{name: "remote without port", spec: "8080 localhost", tunType: TunnelTypeLocal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseForward(tt.tunType, tt.spec, "")
			if ok != tt.wantOK {
				t.Fatalf("parseForward(%q) ok = %v, want %v", tt.spec, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("parseForward(%q) = %+v, want %+v", tt.spec, got, tt.want)
			}
		})
	}
}

func TestHostTitle(t *testing.T) {
	tests := []struct {
		comment string
		alias   string
		count   int
		want    string
	}{
		{"", "web", 1, "web"},
		{"Web tier", "web", 1, "Web tier"},
		{"Web tier", "web", 2, "Web tier (web)"},
	}
	for _, tt := range tests {
		if got := hostTitle(tt.comment, tt.alias, tt.count); got != tt.want {
			t.Errorf("hostTitle(%q, %q, %d) = %q, want %q", tt.comment, tt.alias, tt.count, got, tt.want)
		}
	}
}
