// Package mount defines canonical virtual-resource mappings.
package mount

import (
	"fmt"
	"net/url"
	"strings"
)

// Mount maps a frontend path pattern to a backend URL pattern.
type Mount struct {
	Path     string `json:"path"`
	Target   string `json:"target"`
	Priority int    `json:"priority,omitempty"`
}

var supportedTargetSchemes = map[string]struct{}{
	"delegate": {},
	"ftp":      {},
	"http":     {},
	"https":    {},
}

// Validate checks one mapping without modifying it.
func (m Mount) Validate() error {
	if m.Path == "" {
		return fmt.Errorf("mount path is required")
	}
	if !strings.HasPrefix(m.Path, "/") {
		return fmt.Errorf("mount path %q must be absolute", m.Path)
	}
	if strings.Count(m.Path, "*") > 1 || (strings.Contains(m.Path, "*") && !strings.HasSuffix(m.Path, "*")) {
		return fmt.Errorf("mount path %q has an invalid wildcard", m.Path)
	}
	if m.Target == "" {
		return fmt.Errorf("mount target is required")
	}
	target, err := url.Parse(m.Target)
	if err != nil {
		return fmt.Errorf("mount target: %w", err)
	}
	if _, ok := supportedTargetSchemes[strings.ToLower(target.Scheme)]; !ok {
		return fmt.Errorf("mount target scheme %q is unsupported", target.Scheme)
	}
	if target.Host == "" {
		return fmt.Errorf("mount target %q requires a host", m.Target)
	}
	if strings.Count(target.Path, "*") > 1 || (strings.Contains(target.Path, "*") && !strings.HasSuffix(target.Path, "*")) {
		return fmt.Errorf("mount target %q has an invalid wildcard", m.Target)
	}
	return nil
}
