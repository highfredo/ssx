// Package system wraps OS-specific helpers: launching processes, opening
// files/URLs with the default handler and editing files in $EDITOR.
package system

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/highfredo/ssx/internal/paths"
)

// Start launches a detached process and reaps it in the background so it
// never becomes a zombie. The caller does not wait for it to finish.
func Start(path string, args ...string) error {
	cmd := exec.Command(path, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", path, err)
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			slog.Debug("background process exited", "path", path, "err", err)
		}
	}()
	return nil
}

// Open opens a file path or URL with the platform's default handler.
func Open(uri string) error {
	var opener string
	var target = uri
	if IsDarwin {
		opener = "open"
	} else if IsWSL {
		opener = "/mnt/c/Windows/explorer.exe"
		if strings.Contains(uri, "://") {
			target = uri
		} else {
			target = paths.ToWindows(uri)
		}
	} else if IsWindows {
		opener = "explorer.exe"
	} else {
		opener = "xdg-open"
	}

	cmd := exec.Command(opener, target)
	return cmd.Start()
}

// EditFile opens filePath at the given 1-indexed line in the background
// using $VISUAL, $EDITOR, or a fallback opener/editor.
func EditFile(filePath string, line int) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}

	var parts []string
	if editor != "" {
		parts = strings.Fields(editor)
	}
	if len(parts) == 0 {
		for _, candidate := range []string{"open", "code", "nvim", "vim", "vi", "nano", "notepad"} {
			if p, err := exec.LookPath(candidate); err == nil {
				parts = []string{p}
				break
			}
		}
	}
	if len(parts) == 0 {
		return fmt.Errorf("no editor found (set $EDITOR)")
	}

	bin := parts[0]
	args := parts[1:]

	target := paths.ExpandTilde(filePath)
	if IsWSL && strings.HasSuffix(strings.ToLower(bin), ".exe") {
		target = paths.ToWindows(target)
	}

	name := strings.TrimSuffix(strings.ToLower(filepath.Base(bin)), ".exe")
	if line > 0 {
		switch name {
		case "code", "code-insiders", "codium", "cursor", "windsurf":
			args = append(args, "--goto", fmt.Sprintf("%s:%d", target, line))
		case "subl", "zed":
			args = append(args, fmt.Sprintf("%s:%d", target, line))
		case "vim", "nvim", "vi", "nano", "emacs", "hx", "helix", "micro", "gedit", "kate":
			args = append(args, fmt.Sprintf("+%d", line), target)
		default:
			args = append(args, target)
		}
	} else {
		args = append(args, target)
	}

	return exec.Command(bin, args...).Start()
}
