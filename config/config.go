// Package config defines the canonical, side-effect-free gateway configuration.
package config

import (
	"fmt"
	"strings"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/policy"
)

// Config is the complete configuration consumed by the gateway runtime.
type Config struct {
	Servers  []Server      `json:"servers"`
	Mounts   []mount.Mount `json:"mounts,omitempty"`
	Policies []policy.Rule `json:"policies,omitempty"`
}

// Server describes a named protocol listener.
type Server struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Listen   string `json:"listen"`
}

// Validate reports configuration errors without modifying the receiver.
func (c Config) Validate() error {
	seen := make(map[string]struct{}, len(c.Servers))
	for i, server := range c.Servers {
		if strings.TrimSpace(server.Name) == "" {
			return fmt.Errorf("server %d: name is required", i)
		}
		if strings.TrimSpace(server.Protocol) == "" {
			return fmt.Errorf("server %q: protocol is required", server.Name)
		}
		if strings.TrimSpace(server.Listen) == "" {
			return fmt.Errorf("server %q: listen address is required", server.Name)
		}
		if _, ok := seen[server.Name]; ok {
			return fmt.Errorf("duplicate server name %q", server.Name)
		}
		seen[server.Name] = struct{}{}
	}
	for i, mapping := range c.Mounts {
		if err := mapping.Validate(); err != nil {
			return fmt.Errorf("mount %d: %w", i, err)
		}
	}
	for i, rule := range c.Policies {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("policy %d: %w", i, err)
		}
	}
	return nil
}
