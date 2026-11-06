package ssh

import (
	"errors"
	"testing"
)

func TestClassifyExit(t *testing.T) {
	exit := errors.New("exit status 255")
	tests := []struct {
		name   string
		err    error
		stderr string
		isAuth bool
	}{
		{"nil error", nil, "Permission denied (publickey).", false},
		{"auth failure", exit, "debug1: ...\r\nadmin@h: Permission denied (publickey,password).\n", true},
		{"other failure", exit, "ssh: connect to host h port 22: Connection refused", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyExit(tt.err, tt.stderr)
			if errors.Is(got, ErrAuthFailed) != tt.isAuth {
				t.Fatalf("errors.Is(ErrAuthFailed) = %v, want %v (err=%v)", !tt.isAuth, tt.isAuth, got)
			}
			if tt.err != nil && !errors.Is(got, tt.err) {
				t.Fatalf("original error not wrapped: %v", got)
			}
		})
	}
}

func TestLineWriterSplitsLines(t *testing.T) {
	var lines []string
	w := newLineWriter(func(l string) { lines = append(lines, l) })
	for _, chunk := range []string{"debug1: a\r\nPermission ", "denied\nleft"} {
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if len(lines) != 2 || lines[0] != "debug1: a" || lines[1] != "Permission denied" {
		t.Fatalf("lines = %q", lines)
	}
	if w.String() != "debug1: a\r\nPermission denied\nleft" {
		t.Fatalf("String() = %q", w.String())
	}
}

func TestWithPassword(t *testing.T) {
	h := &HostConfig{Name: "x", PasswordCommand: "pass show x"}
	c := h.WithPassword("s3cret")
	if c.Password != "s3cret" || c.PasswordCommand != "" || c.Name != "x" {
		t.Fatalf("unexpected copy: %+v", c)
	}
	if h.Password != "" || h.PasswordCommand == "" {
		t.Fatalf("original mutated: %+v", h)
	}
}
