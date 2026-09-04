package config

import (
	"reflect"
	"strings"
	"testing"

	"gitea.local/ryan/new-delegate/mount"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "valid HTTP server",
			config: Config{Servers: []Server{{
				Name:     "public",
				Protocol: "http",
				Listen:   ":8080",
			}}},
		},
		{
			name:    "missing name",
			config:  Config{Servers: []Server{{Protocol: "http", Listen: ":8080"}}},
			wantErr: "name",
		},
		{
			name:    "missing protocol",
			config:  Config{Servers: []Server{{Name: "public", Listen: ":8080"}}},
			wantErr: "protocol",
		},
		{
			name:    "missing listen address",
			config:  Config{Servers: []Server{{Name: "public", Protocol: "http"}}},
			wantErr: "listen",
		},
		{
			name: "duplicate names",
			config: Config{Servers: []Server{
				{Name: "public", Protocol: "http", Listen: ":8080"},
				{Name: "public", Protocol: "http", Listen: ":8081"},
			}},
			wantErr: "duplicate server name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := cloneConfig(tt.config)
			err := tt.config.Validate()

			if tt.wantErr == "" && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("Validate() error = %v, want error containing %q", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.config, before) {
				t.Fatalf("Validate() mutated input: got %#v, want %#v", tt.config, before)
			}
		})
	}
}

func cloneConfig(in Config) Config {
	out := in
	out.Servers = append([]Server(nil), in.Servers...)
	out.Mounts = append([]mount.Mount(nil), in.Mounts...)
	return out
}
