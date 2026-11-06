package ssh

import (
	"fmt"
	"net"
	"strconv"
	"testing"
)

func TestIsSSHProcess(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "unix ssh", input: "ssh", want: true},
		{name: "windows ssh", input: "ssh.exe", want: true},
		{name: "uppercase SSH", input: "SSH.EXE", want: true},
		{name: "other process", input: "nginx", want: false},
		{name: "sshd daemon", input: "sshd", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSSHProcess(tt.input); got != tt.want {
				t.Errorf("isSSHProcess(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetTunnelStatus(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	openPort := strconv.Itoa(addr.Port)

	tests := []struct {
		name      string
		tunnel    Tunnel
		wantState TunnelState
		wantErr   bool
	}{
		{
			name:      "invalid port",
			tunnel:    Tunnel{LocalPort: "abc"},
			wantState: TunnelClosed,
			wantErr:   true,
		},
		{
			name:      "port in use by non-ssh test process",
			tunnel:    Tunnel{LocalPort: openPort},
			wantState: TunnelPortInUse,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := GetTunnelStatus(nil, tt.tunnel)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetTunnelStatus(%v) error = %v, wantErr %v", tt.tunnel, err, tt.wantErr)
			}
			if status.State != tt.wantState {
				t.Errorf("GetTunnelStatus(%v).State = %v, want %v (owner=%q, pid=%d)",
					tt.tunnel, status.State, tt.wantState, status.Owner, status.PID)
			}
		})
	}

	// Close the listener and verify the port reports closed.
	_ = ln.Close()
	status, err := GetTunnelStatus(nil, Tunnel{LocalPort: openPort})
	if err != nil {
		t.Fatalf("GetTunnelStatus on closed port error = %v", err)
	}
	if status.State != TunnelClosed {
		t.Errorf("expected TunnelClosed after closing listener, got %v (id=%s)",
			status.State, fmt.Sprint(status.PID))
	}
}

func TestCmdlineMatchesHost(t *testing.T) {
	devHost := &HostConfig{Name: "dev", Hostname: "dev.lan", Port: "22"}

	tests := []struct {
		name    string
		cmdline string
		host    *HostConfig
		want    bool
	}{
		{
			name:    "direct ssh matches exact alias",
			cmdline: "ssh -N -L 8080:localhost:8080 dev",
			host:    devHost,
			want:    true,
		},
		{
			name:    "direct ssh prefix alias does not false-match",
			cmdline: "ssh -N -L 8080:localhost:8080 dev-2",
			host:    devHost,
			want:    false,
		},
		{
			name:    "direct ssh matches exact hostname",
			cmdline: "ssh -N -L 8080:localhost:8080 dev.lan",
			host:    devHost,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cmdlineMatchesHost(tt.cmdline, tt.host); got != tt.want {
				t.Errorf("cmdlineMatchesHost(%q, %v) = %v, want %v", tt.cmdline, tt.host.Name, got, tt.want)
			}
		})
	}
}

func TestParseTunnel(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantNil    bool
		localPort  string
		remoteHost string
		remotePort string
	}{
		{
			name:       "single port shortcut",
			input:      "8080",
			localPort:  "8080",
			remoteHost: "localhost",
			remotePort: "8080",
		},
		{
			name:       "localPort:remotePort shortcut",
			input:      "8080:8081",
			localPort:  "8080",
			remoteHost: "localhost",
			remotePort: "8081",
		},
		{
			name:       "port:remoteHost shortcut",
			input:      "8085:miinternal.net",
			localPort:  "8085",
			remoteHost: "miinternal.net",
			remotePort: "8085",
		},
		{
			name:       "remoteHost:port shortcut",
			input:      "miinternal.net:8085",
			localPort:  "8085",
			remoteHost: "miinternal.net",
			remotePort: "8085",
		},
		{
			name:       "localPort:remoteHost:remotePort",
			input:      "8080:miinternal.net:8085",
			localPort:  "8080",
			remoteHost: "miinternal.net",
			remotePort: "8085",
		},
		{
			name:       "valid ipv6 remote host",
			input:      "9000:[::1]:9001",
			localPort:  "9000",
			remoteHost: "::1",
			remotePort: "9001",
		},
		{
			name:    "invalid local port",
			input:   "99999:localhost:8081",
			wantNil: true,
		},
		{
			name:    "plain search text",
			input:   "web",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTunnel("mini", tt.input)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("ParseTunnel(%q) = %+v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ParseTunnel(%q) = nil, want non-nil", tt.input)
			}
			if got.LocalPort != tt.localPort || got.RemoteHost != tt.remoteHost || got.RemotePort != tt.remotePort {
				t.Errorf("ParseTunnel(%q) = %s:%s:%s, want %s:%s:%s",
					tt.input, got.LocalPort, got.RemoteHost, got.RemotePort,
					tt.localPort, tt.remoteHost, tt.remotePort)
			}
		})
	}
}

func TestTunnelFromCmdline(t *testing.T) {
	tests := []struct {
		name       string
		cmdline    string
		localPort  string
		remoteHost string
		remotePort string
	}{
		{
			name:       "direct ssh with -L",
			cmdline:    "ssh -v -N -F /dev/null -L 8085:miinternal.net:8081 mini",
			localPort:  "8085",
			remoteHost: "miinternal.net",
			remotePort: "8081",
		},
		{
			name:       "controlmaster mux process without -L",
			cmdline:    "ssh: /home/user/.cache/ssx/cm-mini-22 [mux]",
			localPort:  "8085",
			remoteHost: "",
			remotePort: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tunnelFromCmdline("mini", tt.localPort, tt.cmdline)
			if got.LocalPort != tt.localPort || got.RemoteHost != tt.remoteHost || got.RemotePort != tt.remotePort {
				t.Errorf("tunnelFromCmdline(%q) = %s:%s:%s, want %s:%s:%s",
					tt.cmdline, got.LocalPort, got.RemoteHost, got.RemotePort,
					tt.localPort, tt.remoteHost, tt.remotePort)
			}
		})
	}
}
