package mount

import (
	"strings"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr string
	}{
		{name: "ordinary", raw: "/a/b", want: "/a/b"},
		{name: "duplicate separators", raw: "/a//b/", want: "/a/b/"},
		{name: "ordinary escaping", raw: "/a/%62", want: "/a/b"},
		{name: "root", raw: "/", want: "/"},
		{name: "relative", raw: "a/b", wantErr: "absolute"},
		{name: "plain traversal", raw: "/a/../secret", wantErr: "traversal"},
		{name: "encoded traversal", raw: "/%2e%2e/secret", wantErr: "traversal"},
		{name: "double encoded traversal", raw: "/%252e%252e/secret", wantErr: "double encoding"},
		{name: "encoded slash", raw: "/safe%2fsecret", wantErr: "separator"},
		{name: "encoded NUL", raw: "/safe%00secret", wantErr: "control"},
		{name: "backslash", raw: `/safe\..\secret`, wantErr: "backslash"},
		{name: "bad escape", raw: "/%zz", wantErr: "escape"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePath(tt.raw)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("NormalizePath(%q) error = %v, want nil", tt.raw, err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("NormalizePath(%q) error = %v, want error containing %q", tt.raw, err, tt.wantErr)
			}
			if tt.wantErr == "" && got != tt.want {
				t.Fatalf("NormalizePath(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
