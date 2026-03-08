package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create password store dir %s: %w", dir, err)
	}
	b, err := json.Marshal(passwords)
	if err != nil {
		return fmt.Errorf("encode password store: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write password store temp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("commit password store %s: %w", s.path, err)
	}
	return nil
}
