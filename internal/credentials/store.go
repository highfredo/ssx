package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/highfredo/ssx/internal/atomicfile"
)

// Store persists per-host SSH passwords.
type Store struct {
	path string
}

func NewStore() *Store {
	return &Store{path: defaultPath()}
}

func defaultPath() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "ssx", "passwords.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "passwords.json"
	}
	return filepath.Join(home, ".local", "state", "ssx", "passwords.json")
}

func (s *Store) Load() (map[string]string, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read password store %s: %w", s.path, err)
	}
	if len(raw) == 0 {
		return map[string]string{}, nil
	}
	var passwords map[string]string
	if err := json.Unmarshal(raw, &passwords); err != nil {
		return nil, fmt.Errorf("decode password store %s: %w", s.path, err)
	}
	if passwords == nil {
		passwords = map[string]string{}
	}
	return passwords, nil
}

func (s *Store) Save(passwords map[string]string) error {
	if err := atomicfile.WriteJSON(s.path, passwords); err != nil {
		return fmt.Errorf("save password store %s: %w", s.path, err)
	}
	return nil
}
