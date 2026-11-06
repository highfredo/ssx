package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/highfredo/ssx/internal/paths"
	"github.com/highfredo/ssx/internal/system"
)

// ensureAskpassScript writes a static, secret-free helper script into the app
// cache directory that reads either SSX_PASSWORD or SSX_PASS_CMD from the
// child process environment.
func ensureAskpassScript() (string, error) {
	if err := os.MkdirAll(paths.CacheDir(), 0o700); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	name := "askpass.sh"
	script := "#!/bin/sh\nif [ -n \"$SSX_PASS_CMD\" ]; then\n  sh -c \"$SSX_PASS_CMD\"\nelse\n  printf '%s\\n' \"$SSX_PASSWORD\"\nfi\n"
	if system.IsWindows {
		name = "askpass.bat"
		script = "@powershell -NoProfile -NonInteractive -Command \"if ($env:SSX_PASS_CMD) { Invoke-Expression $env:SSX_PASS_CMD } else { Write-Output $env:SSX_PASSWORD }\"\r\n"
	}

	scriptPath := filepath.Join(paths.CacheDir(), name)
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return "", fmt.Errorf("write askpass script: %w", err)
	}
	return scriptPath, nil
}

func configureAskpass(cmd *exec.Cmd, extraEnv ...string) error {
	scriptPath, err := ensureAskpassScript()
	if err != nil {
		return err
	}

	cmd.Env = append(os.Environ(),
		"SSH_ASKPASS="+scriptPath,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=ssx:0",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	return nil
}

// ConfigurePassword wires SSH_ASKPASS-based password auth for an ssh command.
func ConfigurePassword(cmd *exec.Cmd, password string) error {
	if password == "" {
		return nil
	}
	return configureAskpass(cmd, "SSX_PASSWORD="+password)
}

// ConfigurePasswordCommand wires SSH_ASKPASS-based auth using a shell command
// whose stdout provides the password (e.g. "bw get password mysite").
func ConfigurePasswordCommand(cmd *exec.Cmd, passwordCommand string) error {
	if passwordCommand == "" {
		return nil
	}
	return configureAskpass(cmd, "SSX_PASS_CMD="+passwordCommand)
}
