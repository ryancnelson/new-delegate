package mount

import (
	"strings"
	"testing"

	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestMountValidate(t *testing.T) {
	tests := []struct {
		name    string
		mount   Mount
		wantErr string
	}{
		{
			name:  "HTTP wildcard mapping",
			mount: Mount{Path: "/api/*", Target: "http://backend.internal/v1/*"},
		},
		{
			name:  "FTP wildcard mapping",
			mount: Mount{Path: "/files/*", Target: "ftp://files.internal/incoming/*", Priority: 20},
		},
		{
			name:  "delegate chain",
			mount: Mount{Path: "/*", Target: "delegate://next-proxy:8081/*"},
		},
		{
			name:  "absolute HTTP source",
			mount: Mount{Source: "http://Example.COM:8080/docs/*", Target: "https://docs.internal/*"},
		},
		{name: "path and URL source", mount: Mount{Path: "/*", Source: "http://example.com/*", Target: "http://backend/*"}, wantErr: "exactly one"},
		{name: "source userinfo", mount: Mount{Source: "http://user@example.com/*", Target: "http://backend/*"}, wantErr: "userinfo"},
		{name: "source query", mount: Mount{Source: "http://example.com/*?x=1", Target: "http://backend/*"}, wantErr: "query"},
		{name: "source fragment", mount: Mount{Source: "http://example.com/*#x", Target: "http://backend/*"}, wantErr: "fragment"},
		{name: "source encoded separator", mount: Mount{Source: "http://example.com/a%2fb/*", Target: "http://backend/*"}, wantErr: "source path"},
		{name: "source traversal", mount: Mount{Source: "http://example.com/a/../*", Target: "http://backend/*"}, wantErr: "source path"},
		{name: "source duplicate separator", mount: Mount{Source: "http://example.com/a//b/*", Target: "http://backend/*"}, wantErr: "source path"},
		{name: "unsupported source scheme", mount: Mount{Source: "ftp://example.com/*", Target: "http://backend/*"}, wantErr: "source scheme"},
		{
			name: "HTTPS with backend TLS policy",
			mount: Mount{Path: "/*", Target: "https://backend.internal/*", TLS: &tlsconfig.Backend{
				CAFile: "certs/backend-ca.pem", MinimumVersion: "1.3",
			}},
		},
		{
			name:    "backend TLS on plaintext target",
			mount:   Mount{Path: "/*", Target: "http://backend.internal/*", TLS: &tlsconfig.Backend{}},
			wantErr: "HTTPS",
		},
		{
			name: "invalid backend TLS policy",
			mount: Mount{Path: "/*", Target: "https://backend.internal/*", TLS: &tlsconfig.Backend{
				ClientCertificateFile: "certs/client.pem",
			}},
			wantErr: "together",
		},
		{name: "missing path", mount: Mount{Target: "http://backend/"}, wantErr: "path is required"},
		{name: "relative path", mount: Mount{Path: "api/*", Target: "http://backend/*"}, wantErr: "absolute"},
		{name: "embedded source wildcard", mount: Mount{Path: "/api/*/v1", Target: "http://backend/*"}, wantErr: "wildcard"},
		{name: "missing target", mount: Mount{Path: "/api/*"}, wantErr: "target is required"},
		{name: "unsupported target", mount: Mount{Path: "/api/*", Target: "file:///tmp/*"}, wantErr: "scheme"},
		{name: "target without host", mount: Mount{Path: "/api/*", Target: "http:///v1/*"}, wantErr: "host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mount.Validate()
			if tt.wantErr == "" && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("Validate() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
