package config

import (
	"fmt"
	"os"
	"reflect"
)

// ReloadTOMLFile parses and validates a complete canonical file before
// atomically publishing it. Listener topology is fixed for a running process;
// changing it requires a coordinated restart.
func ReloadTOMLFile(store *Store, path string) error {
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
	if !reflect.DeepEqual(store.Snapshot().Servers, candidate.Servers) {
		return fmt.Errorf("listener topology changed; restart required")
	}
	if err := store.Replace(candidate); err != nil {
		return fmt.Errorf("publish reload configuration: %w", err)
	}
	return nil
}
