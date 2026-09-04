package config

import (
	"fmt"
	"sync/atomic"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/policy"
)

// Store publishes immutable, validated configuration snapshots to concurrent
// readers. A failed replacement leaves the previously published snapshot
// untouched.
type Store struct {
	current atomic.Pointer[Config]
}

// NewStore validates and publishes the initial configuration.
func NewStore(initial Config) (*Store, error) {
	store := &Store{}
	if err := store.Replace(initial); err != nil {
		return nil, fmt.Errorf("initial configuration: %w", err)
	}
	return store, nil
}

// Replace validates a private copy before atomically publishing the complete
// candidate configuration.
func (s *Store) Replace(candidate Config) error {
	private := clone(candidate)
	if err := private.Validate(); err != nil {
		return err
	}
	s.current.Store(&private)
	return nil
}

// Snapshot returns a caller-owned copy of the currently published config.
func (s *Store) Snapshot() Config {
	current := s.current.Load()
	if current == nil {
		return Config{}
	}
	return clone(*current)
}

func clone(source Config) Config {
	result := source
	result.Servers = append([]Server(nil), source.Servers...)
	for i := range result.Servers {
		result.Servers[i].TrustedProxies = append([]string(nil), source.Servers[i].TrustedProxies...)
		if source.Servers[i].TLS != nil {
			copied := *source.Servers[i].TLS
			result.Servers[i].TLS = &copied
		}
	}
	result.Mounts = append([]mount.Mount(nil), source.Mounts...)
	for i := range result.Mounts {
		if source.Mounts[i].TLS != nil {
			copied := *source.Mounts[i].TLS
			result.Mounts[i].TLS = &copied
		}
	}
	result.Policies = append([]policy.Rule(nil), source.Policies...)
	return result
}
