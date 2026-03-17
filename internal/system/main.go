package system

import (
	"errors"
	"os"
	"os/exec"

	"github.com/highfredo/ssx/internal/paths"
)

type CmdRunner struct {
	cmd  *exec.Cmd
	done chan error
}

func (r *CmdRunner) Wait() error {
	return <-r.done
}

func (r *CmdRunner) OnExit(fn func(error)) {
	go func() {
		err := <-r.done
		fn(err)
	}()
}

func Run(path string, args ...string) (*CmdRunner, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	cmd := exec.Command(path, args...)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	r := &CmdRunner{
		cmd:  cmd,
		done: make(chan error, 1),
	}

	go func() {
		r.done <- cmd.Wait()
		close(r.done)
	}()

	return r, nil
}

func Open(uri string) error {
	var opener string
	var target = uri
	if IsDarwin {
		opener = "open"
	} else if IsWSL {
		opener = "/mnt/c/Windows/explorer.exe"
		target = paths.ToWindows(uri)
	} else if IsWindows {
		opener = "explorer.exe"
	} else {
		opener = "xdg-open"
	}

	_, err := Run(opener, target)

	return err
}
