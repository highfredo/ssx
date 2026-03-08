package tunnel

import "testing"

func TestParseSSOwnerLine(t *testing.T) {
	line := `LISTEN 0 128 127.0.0.1:8080 0.0.0.0:* users:(("python3",pid=4321,fd=7))`
	owner, err := parseSSOwnerLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner == nil {
		t.Fatalf("expected owner")
	}
	if owner.PID != 4321 {
		t.Fatalf("pid mismatch: got %d", owner.PID)
	}
	if owner.Command != "python3" {
		t.Fatalf("command mismatch: got %q", owner.Command)
	}
}

func TestParseSSOwnerLineNoUsers(t *testing.T) {
	owner, err := parseSSOwnerLine(`LISTEN 0 128 127.0.0.1:8080 0.0.0.0:*`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != nil {
		t.Fatalf("expected no owner, got %+v", owner)
	}
}
