package ssh

import "testing"

func TestParseConnection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *HostConfig
		wantNil bool
	}{
		{
			name:  "host only",
			input: "example.com",
			want:  &HostConfig{Name: "example.com", Hostname: "example.com", Port: "22"},
		},
		{
			name:  "user and port",
			input: "deploy@10.0.0.1:2222",
			want:  &HostConfig{Name: "10.0.0.1", Hostname: "10.0.0.1", User: "deploy", Port: "2222"},
		},
		{
			name:  "user with password",
			input: "root:s3cr3t@box",
			want:  &HostConfig{Name: "box", Hostname: "box", User: "root", Port: "22", Password: "s3cr3t"},
		},
		{
			name:  "ipv6 with port",
			input: "[::1]:2200",
			want:  &HostConfig{Name: "::1", Hostname: "::1", Port: "2200"},
		},
		{name: "empty", input: "   ", wantNil: true},
		{name: "invalid port", input: "box:99999", wantNil: true},
		{name: "invalid user chars", input: "bad user@box", wantNil: true},
		{name: "invalid host chars", input: "ho st", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseConnection(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("ParseConnection(%q) = %+v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ParseConnection(%q) = nil, want %+v", tt.input, tt.want)
			}
			if !got.IsAdHoc() {
				t.Errorf("ParseConnection(%q).IsAdHoc() = false, want true", tt.input)
			}
			if got.Name != tt.want.Name || got.Hostname != tt.want.Hostname || got.User != tt.want.User ||
				got.Port != tt.want.Port || got.Password != tt.want.Password {
				t.Errorf("ParseConnection(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}
