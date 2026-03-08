package tunnel

import (
	"slices"
	"testing"

	"github.com/highfredo/ssx/internal/sshconfig"
)

func TestBuildSSHArgsDisablesMultiplexing(t *testing.T) {
	args := buildSSHArgs("mini", sshconfig.Tunnel{
		Type:       sshconfig.TunnelTypeLocal,
		LocalPort:  "8080",
		RemoteHost: "localhost",
		RemotePort: "8080",
	}, nil)

	if !slices.Contains(args, "ControlMaster=no") {
		t.Fatalf("expected ControlMaster=no in args: %v", args)
	}
	if !slices.Contains(args, "-F") || !slices.Contains(args, "/dev/null") {
		t.Fatalf("expected '-F /dev/null' in args: %v", args)
	}
	if slices.Contains(args, "ClearAllForwardings=yes") {
		t.Fatalf("did not expect ClearAllForwardings=yes in args: %v", args)
	}
}

func TestBuildSSHArgsUsesResolvedTarget(t *testing.T) {
	target := &sshOpenTarget{
		HostName:      "192.168.10.2",
		User:          "highfredo",
		Port:          "2222",
		ProxyJump:     "bastion",
		IdentityFiles: []string{"/tmp/id_test"},
	}
	args := buildSSHArgs("mini", sshconfig.Tunnel{
		Type:      sshconfig.TunnelTypeDynamic,
		LocalPort: "1080",
	}, target)

	if !slices.Contains(args, "192.168.10.2") {
		t.Fatalf("expected destination hostname in args: %v", args)
	}
	if !slices.Contains(args, "highfredo") || !slices.Contains(args, "2222") {
		t.Fatalf("expected user/port in args: %v", args)
	}
	if !slices.Contains(args, "bastion") || !slices.Contains(args, "/tmp/id_test") {
		t.Fatalf("expected proxyjump and identity in args: %v", args)
	}
}
