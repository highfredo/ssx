// Package atomicfile provides atomic file-write helpers used to safely persist
// JSON state without leaving a half-written file on crash or power loss.
package atomicfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteJSON atomically marshals v to JSON and writes it to path.
// The parent directory is created if it does not already exist.
// The write is crash-safe: data goes to a temp file first, then renamed.
func WriteJSON(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("commit %s: %w", path, err)
	}
	return nil
}
