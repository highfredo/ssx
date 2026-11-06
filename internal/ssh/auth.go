package ssh

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrAuthFailed reports that ssh rejected every offered credential. Callers
// may ask the user for a password and retry with WithPassword.
var ErrAuthFailed = errors.New("authentication failed")

// authFailureMarker is what OpenSSH prints when all auth methods failed.
const authFailureMarker = "Permission denied"

// classifyExit wraps a non-nil ssh exit error as ErrAuthFailed when the
// captured stderr shows an authentication failure.
func classifyExit(err error, stderr string) error {
	if err != nil && strings.Contains(stderr, authFailureMarker) {
		return fmt.Errorf("%w: %w", ErrAuthFailed, err)
	}
	return err
}

// WithPassword returns a shallow copy of h that authenticates with password,
// overriding any configured password or password command.
func (h *HostConfig) WithPassword(password string) *HostConfig {
	c := *h
	c.Password = password
	c.PasswordCommand = ""
	return &c
}

// lineWriter is an io.Writer that splits incoming data into lines and hands
// each complete line to fn. It is safe for concurrent writes and keeps the
// full text for later inspection.
type lineWriter struct {
	mu   sync.Mutex
	all  strings.Builder
	part string
	fn   func(line string)
}

func newLineWriter(fn func(line string)) *lineWriter {
	return &lineWriter{fn: fn}
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.all.Write(p)
	w.part += string(p)
	for {
		nl := strings.IndexByte(w.part, '\n')
		if nl < 0 {
			return len(p), nil
		}
		w.fn(strings.TrimRight(w.part[:nl], "\r"))
		w.part = w.part[nl+1:]
	}
}

// String returns everything written so far.
func (w *lineWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.all.String()
}
