package ssh

import (
	"log/slog"
	"os/exec"
)

// Shell builds an interactive ssh session command for host, with any
// configured forwardings disabled so they do not collide with ssx tunnels.
func Shell(host *HostConfig) *exec.Cmd {
	return PrepareCmd(host, "-o", "ClearAllForwardings=yes")
}

// PrepareCmd builds an ssh command for host with no remote command.
// Extra flags in args are placed before the common options.
func PrepareCmd(host *HostConfig, args ...string) *exec.Cmd {
	return PrepareCmdWithCommand(host, args, nil)
}

// PrepareCmdWithCommand builds an ssh command for host that runs command on
// the remote side. Password authentication, when configured, is wired through
// SSH_ASKPASS so the user is never prompted.
func PrepareCmdWithCommand(host *HostConfig, args []string, command []string) *exec.Cmd {
	cmd := exec.Command("ssh", append(args, "-o", "StrictHostKeyChecking=accept-new")...)

	if err := configureAuth(cmd, host); err != nil {
		slog.Warn("password auth unavailable, ssh may prompt interactively", "host", host.Name, "err", err)
	}

	target := host.Name
	if target == "" {
		target = host.Hostname
	}
	if host.IsAdHoc() {
		// Unknown to ssh's own config: pass connection details explicitly.
		if host.User != "" {
			cmd.Args = append(cmd.Args, "-l", host.User)
		}
		if host.Port != "" && host.Port != DefaultPort {
			cmd.Args = append(cmd.Args, "-p", host.Port)
		}
	}
	cmd.Args = append(cmd.Args, target)
	cmd.Args = append(cmd.Args, command...)

	slog.Debug("ssh command prepared", "args", cmd.Args)
	return cmd
}

// configureAuth enables askpass-based password authentication when the host
// defines a password or a password command.
func configureAuth(cmd *exec.Cmd, host *HostConfig) error {
	var err error
	switch {
	case host.Password != "":
		err = ConfigurePassword(cmd, host.Password)
	case host.PasswordCommand != "":
		err = ConfigurePasswordCommand(cmd, host.PasswordCommand)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	cmd.Args = append(cmd.Args, "-o", "NumberOfPasswordPrompts=1")
	return nil
}
