package sshconfig

import "testing"

func TestParseHostTags(t *testing.T) {
	text := `
Host mini
    #>tags: minipc, personal
    HostName 192.168.10.2

Host prod ops
    #>tags: work, critical
    Port 22
`
	got := parseHostTags(text)

	assertTags(t, got["mini"], []string{"minipc", "personal"})
	assertTags(t, got["prod"], []string{"work", "critical"})
	assertTags(t, got["ops"], []string{"work", "critical"})
}

func TestParseHostTagsIgnoresUnsupportedLocations(t *testing.T) {
	text := `
#>tags: global, ignored
Host *
    #>tags: wildcard, ignored

Match host mini
    #>tags: ignored

Host mini
    #>tags: personal, personal
`
	got := parseHostTags(text)

	assertTags(t, got["mini"], []string{"personal"})
	if _, ok := got["*"]; ok {
		t.Fatalf("unexpected wildcard tags entry")
	}
}

func assertTags(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("value mismatch at %d: got %q want %q", i, got[i], want[i])
		}
	}
}
