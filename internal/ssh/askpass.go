package ssh

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/highfredo/ssx/internal/system"
)

// configureAskpass creates a temporary askpass script with the given content,
// sets SSH_ASKPASS / SSH_ASKPASS_REQUIRE on cmd, and returns a cleanup func.
func configureAskpass(cmd *exec.Cmd, script string, extraEnv ...string) (func(), error) {
	pattern := "ssx-askpass-*"
	if system.IsWindows {
		pattern += ".bat"
	}

	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return nil, fmt.Errorf("create askpass script: %w", err)
	}
	scriptPath := f.Name()
	cleanup := func() { _ = os.Remove(scriptPath) }

	if _, err := f.WriteString(script); err != nil {
		_ = f.Close()
		cleanup()
		return nil, fmt.Errorf("write askpass script: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return nil, fmt.Errorf("close askpass script: %w", err)
	}
	if !system.IsWindows {
		if err := os.Chmod(scriptPath, 0o700); err != nil {
			cleanup()
			return nil, fmt.Errorf("chmod askpass script: %w", err)
		}
	}

	cmd.Env = append(os.Environ(),
		"SSH_ASKPASS="+scriptPath,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=ssx:0",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	return cleanup, nil
}

// ConfigurePassword wires SSH_ASKPASS-based password auth for an ssh command.
func ConfigurePassword(cmd *exec.Cmd, password string) (func(), error) {
	if password == "" {
		return func() {}, nil
	}
	var script string
	if system.IsWindows {
		script = "@powershell -NoProfile -NonInteractive -Command \"Write-Output $env:SSX_PASSWORD\"\r\n"
	} else {
		script = "#!/bin/sh\nprintf '%s\\n' \"$SSX_PASSWORD\"\n"
	}
	return configureAskpass(cmd, script, "SSX_PASSWORD="+password)
}

// ConfigurePasswordCommand wires SSH_ASKPASS-based auth using a shell command
// whose stdout provides the password (e.g. "bw get password mysite").
func ConfigurePasswordCommand(cmd *exec.Cmd, passwordCommand string) (func(), error) {
	if passwordCommand == "" {
		return func() {}, nil
	}
	var script string
	if system.IsWindows {
		script = "@cmd /c " + passwordCommand + "\r\n"
	} else {
		script = "#!/bin/sh\n" + passwordCommand + "\n"
	}
	return configureAskpass(cmd, script)
}
