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
