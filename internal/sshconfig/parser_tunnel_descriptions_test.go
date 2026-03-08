package sshconfig

import "testing"

func TestParseHostTunnelDescriptions(t *testing.T) {
	text := `
Host mini
    LocalForward 8080 localhost:8080 # web server
    RemoteForward 9090 localhost:9090 # remote app
    DynamicForward 1080 # socks

Host prod backup
    LocalForward 5432 db.internal:5432 # postgres
`

	got := parseHostTunnelDescriptions(text)

	assertTunnelDescription(t, got["mini"], Tunnel{
		Type:       TunnelTypeLocal,
		LocalPort:  "8080",
		RemoteHost: "localhost",
		RemotePort: "8080",
	}, "web server")
	assertTunnelDescription(t, got["mini"], Tunnel{
		Type:       TunnelTypeRemote,
		LocalPort:  "9090",
		RemoteHost: "localhost",
		RemotePort: "9090",
	}, "remote app")
	assertTunnelDescription(t, got["mini"], Tunnel{
		Type:      TunnelTypeDynamic,
		LocalPort: "1080",
	}, "socks")
	assertTunnelDescription(t, got["prod"], Tunnel{
		Type:       TunnelTypeLocal,
		LocalPort:  "5432",
		RemoteHost: "db.internal",
		RemotePort: "5432",
	}, "postgres")
	assertTunnelDescription(t, got["backup"], Tunnel{
		Type:       TunnelTypeLocal,
		LocalPort:  "5432",
		RemoteHost: "db.internal",
		RemotePort: "5432",
	}, "postgres")
}

func TestParseHostTunnelDescriptionsIgnoresInvalidLines(t *testing.T) {
	text := `
Host mini
    LocalForward 8080 localhost:8080
    # LocalForward 9090 localhost:9090 # ignored
    Match host mini
    LocalForward 1111 localhost:1111 # ignored
`
	got := parseHostTunnelDescriptions(text)
	if got["mini"] != nil {
		t.Fatalf("expected no tunnel descriptions, got %v", got["mini"])
	}
}

func assertTunnelDescription(t *testing.T, byKey map[string]string, tunnel Tunnel, want string) {
	t.Helper()
	key := tunnelDescriptorKey(tunnel)
	got, ok := byKey[key]
	if !ok {
		t.Fatalf("missing description for %s", key)
	}
	if got != want {
		t.Fatalf("description mismatch for %s: got %q want %q", key, got, want)
	}
}
