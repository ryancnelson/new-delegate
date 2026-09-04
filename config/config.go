// Package config defines the canonical, side-effect-free gateway configuration.
package config

import (
	"fmt"
	"strings"
)

// Config is the complete configuration consumed by the gateway runtime.
type Config struct {
	Servers []Server `json:"servers"`
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
	return nil
}
