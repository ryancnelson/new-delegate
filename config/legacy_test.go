package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gitea.local/ryan/new-delegate/mount"
)

func TestParseLegacyArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    Config
		wantErr string
	}{
		{
			name: "protocol default port",
			args: []string{"SERVER=http"},
			want: Config{Servers: []Server{{Name: "default", Protocol: "http", Listen: ":80"}}},
		},
		{
			name: "port before server",
			args: []string{"-P8080", "SERVER=http"},
			want: Config{Servers: []Server{{Name: "default", Protocol: "http", Listen: ":8080"}}},
		},
		{
			name: "separate port and whitespace",
			args: []string{" SERVER=HTTP ", " -P ", " 8081 "},
			want: Config{Servers: []Server{{Name: "default", Protocol: "http", Listen: ":8081"}}},
		},
		{
			name: "identical repetitions are idempotent",
			args: []string{"SERVER=http", "SERVER=http", "-P8080", "-P8080"},
			want: Config{Servers: []Server{{Name: "default", Protocol: "http", Listen: ":8080"}}},
		},
		{name: "missing server", args: []string{"-P8080"}, wantErr: "SERVER"},
		{name: "conflicting servers", args: []string{"SERVER=http", "SERVER=ftp"}, wantErr: "conflicting SERVER"},
		{name: "conflicting ports", args: []string{"SERVER=http", "-P8080", "-P8081"}, wantErr: "conflicting -P"},
		{name: "invalid port text", args: []string{"SERVER=http", "-Pnope"}, wantErr: "invalid port"},
		{name: "invalid port zero", args: []string{"SERVER=http", "-P0"}, wantErr: "invalid port"},
		{name: "invalid port too large", args: []string{"SERVER=http", "-P65536"}, wantErr: "invalid port"},
		{name: "unknown directive", args: []string{"SERVER=http", "BOGUS=yes"}, wantErr: "unknown directive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := append([]string(nil), tt.args...)
			got, err := ParseLegacyArgs(tt.args)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("ParseLegacyArgs() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("ParseLegacyArgs() error = %v, want error containing %q", err, tt.wantErr)
			}
			if tt.wantErr == "" && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseLegacyArgs() = %#v, want %#v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.args, before) {
				t.Fatalf("ParseLegacyArgs() mutated arguments: got %#v, want %#v", tt.args, before)
			}
		})
	}
}

func TestParseLegacyMounts(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []mount.Mount
		wantErr string
	}{
		{
			name: "quoted wildcard mount",
			args: []string{"SERVER=http", `MOUNT="/files/* ftp://files.internal/incoming/*"`},
			want: []mount.Mount{{Path: "/files/*", Target: "ftp://files.internal/incoming/*"}},
		},
		{
			name: "multiple mounts retain declaration order",
			args: []string{
				`MOUNT="/api/* http://api.internal/v1/* priority=20"`,
				"SERVER=http",
				"MOUNT=/* http://web.internal/*",
			},
			want: []mount.Mount{
				{Path: "/api/*", Target: "http://api.internal/v1/*", Priority: 20},
				{Path: "/*", Target: "http://web.internal/*"},
			},
		},
		{name: "missing target", args: []string{"SERVER=http", "MOUNT=/api/*"}, wantErr: "MOUNT"},
		{name: "unknown option", args: []string{"SERVER=http", "MOUNT=/api/* http://backend/* cache=yes"}, wantErr: "unknown MOUNT option"},
		{name: "invalid priority", args: []string{"SERVER=http", "MOUNT=/api/* http://backend/* priority=high"}, wantErr: "priority"},
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
			if tt.wantErr == "" && !reflect.DeepEqual(got.Mounts, tt.want) {
				t.Fatalf("ParseLegacyArgs().Mounts = %#v, want %#v", got.Mounts, tt.want)
			}
		})
	}
}

func TestParseLegacyArgsGolden(t *testing.T) {
	got, err := ParseLegacyArgs([]string{"-P8080", "SERVER=http"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')

	want, err := os.ReadFile(filepath.Join("testdata", "server.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != string(want) {
		t.Fatalf("canonical config mismatch\ngot:\n%s\nwant:\n%s", encoded, want)
	}
}
