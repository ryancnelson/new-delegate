package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gitea.local/ryan/new-delegate/policy"
)

func TestParseLegacyPolicy(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []policy.Rule
		wantErr string
	}{
		{
			name: "permit and fallback reject",
			args: []string{
				"SERVER=http",
				`PERMIT="*:*:192.168.1.0/24"`,
				`PERMIT="http:*.internal.com:*"`,
				`REJECT="*:*:*"`,
			},
			want: []policy.Rule{
				{Effect: policy.Permit, Protocol: "*", Destination: "*", Source: "192.168.1.0/24", Priority: 0},
				{Effect: policy.Permit, Protocol: "http", Destination: "*.internal.com", Source: "*", Priority: -1},
				{Effect: policy.Reject, Protocol: "*", Destination: "*", Source: "*", Priority: -2},
			},
		},
		{
			name: "surrounding whitespace",
			args: []string{"SERVER=http", " PERMIT= http : api.internal : 127.0.0.1 "},
			want: []policy.Rule{{Effect: policy.Permit, Protocol: "http", Destination: "api.internal", Source: "127.0.0.1"}},
		},
		{name: "missing selector", args: []string{"SERVER=http", "PERMIT=http:*"}, wantErr: "three selectors"},
		{name: "empty selector", args: []string{"SERVER=http", "REJECT=http::127.0.0.1"}, wantErr: "empty selector"},
		{name: "invalid source CIDR", args: []string{"SERVER=http", "PERMIT=http:*:192.168.1.500/24"}, wantErr: "source"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLegacyArgs(tt.args)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("ParseLegacyArgs() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("ParseLegacyArgs() error = %v, want error containing %q", err, tt.wantErr)
			}
			if tt.wantErr == "" && !reflect.DeepEqual(got.Policies, tt.want) {
				t.Fatalf("ParseLegacyArgs().Policies = %#v, want %#v", got.Policies, tt.want)
			}
		})
	}
}

func TestParseLegacyPolicyGolden(t *testing.T) {
	got, err := ParseLegacyArgs([]string{
		"SERVER=http",
		`PERMIT="http:*.internal.com:10.0.0.0/8"`,
		`REJECT="*:*:*"`,
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(got.Policies, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	want, err := os.ReadFile(filepath.Join("testdata", "policy.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != string(want) {
		t.Fatalf("canonical policy mismatch\ngot:\n%s\nwant:\n%s", encoded, want)
	}
}
