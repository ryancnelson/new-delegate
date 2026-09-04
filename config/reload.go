package config

import (
	"fmt"
	"os"

	"gitea.local/ryan/new-delegate/tlsconfig"
)

// ReloadTOMLFile parses and validates a complete canonical file before
// atomically publishing it. Listener topology is fixed for a running process;
// changing it requires a coordinated restart.
func ReloadTOMLFile(store *Store, path string) error {
	return ReloadTOMLFileWithValidation(store, path, nil)
}

// ReloadTOMLFileWithValidation applies a runtime capability check after
// canonical validation but before topology comparison or publication.
func ReloadTOMLFileWithValidation(store *Store, path string, validate func(Config) error) error {
	if store == nil {
		return fmt.Errorf("configuration store is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open reload configuration %q: %w", path, err)
	}
	candidate, parseErr := ParseTOML(file)
	closeErr := file.Close()
	if parseErr != nil {
		return parseErr
	}
	if closeErr != nil {
		return fmt.Errorf("close reload configuration %q: %w", path, closeErr)
	}
	if validate != nil {
		if err := validate(candidate); err != nil {
			return err
		}
	}
	current := store.Snapshot()
	if !sameListenerTopology(current.Servers, candidate.Servers) {
		return fmt.Errorf("listener topology changed; restart required")
	}
	if !sameBackendTLSPolicies(current, candidate) {
		return fmt.Errorf("backend TLS policy changed; restart required")
	}
	if err := store.Replace(candidate); err != nil {
		return fmt.Errorf("publish reload configuration: %w", err)
	}
	return nil
}

func sameBackendTLSPolicies(current, candidate Config) bool {
	collect := func(configured Config) map[tlsconfig.Backend]struct{} {
		policies := make(map[tlsconfig.Backend]struct{})
		for _, mounted := range configured.Mounts {
			if mounted.TLS != nil {
				policies[*mounted.TLS] = struct{}{}
			}
		}
		return policies
	}
	currentPolicies := collect(current)
	candidatePolicies := collect(candidate)
	if len(currentPolicies) != len(candidatePolicies) {
		return false
	}
	for policy := range currentPolicies {
		if _, ok := candidatePolicies[policy]; !ok {
			return false
		}
	}
	return true
}

func sameListenerTopology(current, candidate []Server) bool {
	if len(current) != len(candidate) {
		return false
	}
	listeners := make(map[string]Server, len(current))
	for _, server := range current {
		listeners[server.Name] = server
	}
	for _, server := range candidate {
		existing, ok := listeners[server.Name]
		if !ok || existing.Protocol != server.Protocol || existing.Listen != server.Listen || !sameFrontendTLS(existing, server) {
			return false
		}
	}
	return true
}

func sameFrontendTLS(current, candidate Server) bool {
	if current.TLS == nil || candidate.TLS == nil {
		return current.TLS == nil && candidate.TLS == nil
	}
	return *current.TLS == *candidate.TLS
}
