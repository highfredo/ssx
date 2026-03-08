package sshpass

import (
	"fmt"
	"os"
	"os/exec"
)

// Configure wires SSH_ASKPASS-based password auth for an ssh command.
// It creates a temporary askpass script and returns a cleanup function.
func Configure(cmd *exec.Cmd, password string) (func(), error) {
	if password == "" {
		return func() {}, nil
	}

	f, err := os.CreateTemp("", "ssx-askpass-*")
	if err != nil {
		return nil, fmt.Errorf("create askpass script: %w", err)
	}
	scriptPath := f.Name()
	script := "#!/bin/sh\nprintf '%s\\n' \"$SSX_PASSWORD\"\n"
	if _, err := f.WriteString(script); err != nil {
		_ = f.Close()
		_ = os.Remove(scriptPath)
		return nil, fmt.Errorf("write askpass script: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(scriptPath)
		return nil, fmt.Errorf("close askpass script: %w", err)
	}
	if err := os.Chmod(scriptPath, 0o700); err != nil {
		_ = os.Remove(scriptPath)
		return nil, fmt.Errorf("chmod askpass script: %w", err)
	}

	cmd.Env = append(os.Environ(),
		"SSH_ASKPASS="+scriptPath,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=ssx:0",
		"SSX_PASSWORD="+password,
	)

	cleanup := func() {
		_ = os.Remove(scriptPath)
	}
	return cleanup, nil
}
