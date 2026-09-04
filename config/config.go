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
	Servers  []Server      `json:"servers" toml:"servers"`
	Mounts   []mount.Mount `json:"mounts,omitempty" toml:"mounts"`
	Policies []policy.Rule `json:"policies,omitempty" toml:"policies"`
}

// Server describes a named protocol listener.
type Server struct {
	Name     string `json:"name" toml:"name"`
	Protocol string `json:"protocol" toml:"protocol"`
	Listen   string `json:"listen" toml:"listen"`
}

// Validate reports configuration errors without modifying the receiver.
func (c Config) Validate() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("at least one server is required")
	}
	seen := make(map[string]struct{}, len(c.Servers))
	seenListen := make(map[string]struct{}, len(c.Servers))
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
		if _, ok := seenListen[server.Listen]; ok {
			return fmt.Errorf("duplicate listen address %q", server.Listen)
		}
		seenListen[server.Listen] = struct{}{}
	}
	for i, mapping := range c.Mounts {
		if err := mapping.Validate(); err != nil {
			return fmt.Errorf("mount %d: %w", i, err)
		}
		var scopedServer *Server
		if mapping.Server != "" && mapping.Server != "*" {
			for serverIndex := range c.Servers {
				if c.Servers[serverIndex].Name == mapping.Server {
					scopedServer = &c.Servers[serverIndex]
					break
				}
			}
			if scopedServer == nil {
				return fmt.Errorf("mount %d: unknown server %q", i, mapping.Server)
			}
		}
		if mapping.Protocol != "" && mapping.Protocol != "*" {
			if scopedServer != nil && !strings.EqualFold(scopedServer.Protocol, mapping.Protocol) {
				return fmt.Errorf("mount %d: protocol %q does not match server %q protocol %q", i, mapping.Protocol, scopedServer.Name, scopedServer.Protocol)
			}
			if scopedServer == nil {
				foundProtocol := false
				for _, server := range c.Servers {
					foundProtocol = foundProtocol || strings.EqualFold(server.Protocol, mapping.Protocol)
				}
				if !foundProtocol {
					return fmt.Errorf("mount %d: protocol scope %q matches no server", i, mapping.Protocol)
				}
			}
		}
	}
	for i, rule := range c.Policies {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("policy %d: %w", i, err)
		}
	}
	return nil
}
