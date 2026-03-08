package opener

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"
)

func Open(path string, args ...string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return err
	}

	cmd := exec.Command(path, args...)
	if err := cmd.Start(); err != nil {
		slog.Error("failed to open", "path", path, "args", args)
		return nil
	}
	go func() { _ = cmd.Wait() }()

	return nil
}
