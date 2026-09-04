package mount

import (
	"errors"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name       string
		mounts     []Mount
		path       string
		wantTarget string
		wantErr    error
	}{
		{
			name:       "exact match",
			mounts:     []Mount{{Path: "/health", Target: "http://backend/ready"}},
			path:       "/health",
			wantTarget: "http://backend/ready",
		},
		{
			name: "more specific wildcard wins",
			mounts: []Mount{
				{Path: "/*", Target: "http://fallback/*", Priority: 100},
				{Path: "/api/*", Target: "http://api/v1/*"},
			},
			path:       "/api/users",
			wantTarget: "http://api/v1/users",
		},
		{
			name: "priority breaks equal specificity",
			mounts: []Mount{
				{Path: "/api/*", Target: "http://old/*", Priority: 10},
				{Path: "/api/*", Target: "http://new/*", Priority: 20},
			},
			path:       "/api/users",
			wantTarget: "http://new/users",
		},
		{
			name:       "fallback",
			mounts:     []Mount{{Path: "/*", Target: "http://fallback/root/*"}},
			path:       "/docs/index.html",
			wantTarget: "http://fallback/root/docs/index.html",
		},
		{
			name:    "no match",
			mounts:  []Mount{{Path: "/api/*", Target: "http://api/*"}},
			path:    "/docs",
			wantErr: ErrNoMatch,
		},
		{
			name: "ambiguous winner",
			mounts: []Mount{
				{Path: "/api/*", Target: "http://one/*", Priority: 10},
				{Path: "/api/*", Target: "http://two/*", Priority: 10},
			},
			path:    "/api/users",
			wantErr: ErrAmbiguous,
		},
		{
			name:       "normalizes before matching",
			mounts:     []Mount{{Path: "/api/*", Target: "http://api/*"}},
			path:       "/api//users",
			wantTarget: "http://api/users",
		},
		{
			name:    "unsafe path",
			mounts:  []Mount{{Path: "/*", Target: "http://backend/*"}},
			path:    "/../secret",
			wantErr: ErrUnsafePath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.mounts, tt.path)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Resolve() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() error = %v, want nil", err)
			}
			if got.Target != tt.wantTarget {
				t.Fatalf("Resolve().Target = %q, want %q", got.Target, tt.wantTarget)
			}
		})
	}
}

func TestResolveForScopesMountsToFrontend(t *testing.T) {
	mounts := []Mount{
		{Path: "/api/*", Target: "http://generic/*"},
		{Path: "/api/*", Target: "http://admin/*", Server: "admin"},
		{Path: "/api/*", Target: "http://public/*", Server: "public", Protocol: "http"},
		{Path: "/api/*", Target: "ftp://public/*", Server: "public", Protocol: "ftp"},
	}
	tests := []struct {
		name   string
		server string
		proto  string
		target string
	}{
		{name: "server and protocol scope wins", server: "public", proto: "HTTP", target: "http://public/users"},
		{name: "other named server", server: "admin", proto: "http", target: "http://admin/users"},
		{name: "generic fallback", server: "other", proto: "http", target: "http://generic/users"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveFor(mounts, Request{Path: "/api/users", Server: tt.server, Protocol: tt.proto})
			if err != nil {
				t.Fatalf("ResolveFor() error = %v", err)
			}
			if got.Target != tt.target {
				t.Fatalf("ResolveFor().Target = %q, want %q", got.Target, tt.target)
			}
		})
	}
}

func TestResolveForMatchesURLAuthorityWithoutDNS(t *testing.T) {
	mounts := []Mount{
		{Path: "/docs/*", Target: "http://fallback/*"},
		{Source: "http://Example.COM:8080/docs/*", Target: "http://authority/*"},
	}
	got, err := ResolveFor(mounts, Request{
		Path: "/docs/guide", Scheme: "HTTP", Authority: "example.com:8080",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != "http://authority/guide" || got.Mount.Source == "" {
		t.Fatalf("ResolveFor() = %#v, want URL-authority mount", got)
	}

	got, err = ResolveFor(mounts, Request{
		Path: "/docs/guide", Scheme: "http", Authority: "example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != "http://fallback/guide" {
		t.Fatalf("port-mismatched target = %q, want path fallback", got.Target)
	}
}

func TestResolveForMatchesCONNECTAuthority(t *testing.T) {
	got, err := ResolveFor(
		[]Mount{{Source: "connect://Example.COM:443/", Target: "tcp://127.0.0.1:8443"}},
		Request{Path: "/", Scheme: "CONNECT", Authority: "example.com:443", Protocol: "http"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != "tcp://127.0.0.1:8443" {
		t.Fatalf("ResolveFor().Target = %q", got.Target)
	}
}
