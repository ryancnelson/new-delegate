// Package mount defines canonical virtual-resource mappings.
package mount

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"gitea.local/ryan/new-delegate/tlsconfig"
)

// Mount maps a frontend path or absolute URL pattern to a backend URL pattern.
type Mount struct {
	Path     string             `json:"path,omitempty" toml:"path"`
	Source   string             `json:"source,omitempty" toml:"source"`
	Target   string             `json:"target" toml:"target"`
	Priority int                `json:"priority,omitempty" toml:"priority"`
	Server   string             `json:"server,omitempty" toml:"server"`
	Protocol string             `json:"protocol,omitempty" toml:"protocol"`
	TLS      *tlsconfig.Backend `json:"tls,omitempty" toml:"tls"`
}

var supportedTargetSchemes = map[string]struct{}{
	"delegate": {},
	"ftp":      {},
	"http":     {},
	"https":    {},
	"tcp":      {},
}

// Validate checks one mapping without modifying it.
func (m Mount) Validate() error {
	if strings.ContainsAny(m.Server, " \t\r\n") {
		return fmt.Errorf("mount server scope %q contains whitespace", m.Server)
	}
	if strings.ContainsAny(m.Protocol, " \t\r\n") {
		return fmt.Errorf("mount protocol scope %q contains whitespace", m.Protocol)
	}
	if m.Path == "" && m.Source == "" {
		return fmt.Errorf("mount path is required when URL source is absent")
	}
	if m.Path != "" && m.Source != "" {
		return fmt.Errorf("mount requires exactly one path or absolute URL source")
	}
	sourcePath := m.Path
	if m.Source != "" {
		parsed, err := parseSourceURL(m.Source)
		if err != nil {
			return err
		}
		sourcePath = parsed.EscapedPath()
		if sourcePath == "" {
			sourcePath = "/"
		}
		if strings.EqualFold(parsed.Scheme, "connect") {
			if err := ValidateConnectAuthority(parsed.Host); err != nil {
				return fmt.Errorf("mount CONNECT source: %w", err)
			}
			if sourcePath != "/" {
				return fmt.Errorf("mount CONNECT source path must be /")
			}
		}
		if strings.Contains(sourcePath, "%") {
			return fmt.Errorf("mount source path %q uses ambiguous escaping", sourcePath)
		}
		normalized, err := NormalizePath(sourcePath)
		if err != nil || normalized != sourcePath {
			return fmt.Errorf("mount source path %q is not normalized", sourcePath)
		}
	}
	if !strings.HasPrefix(sourcePath, "/") {
		return fmt.Errorf("mount path %q must be absolute", sourcePath)
	}
	if strings.Count(sourcePath, "*") > 1 || (strings.Contains(sourcePath, "*") && !strings.HasSuffix(sourcePath, "*")) {
		return fmt.Errorf("mount path %q has an invalid wildcard", sourcePath)
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
	if strings.EqualFold(target.Scheme, "tcp") {
		if target.User != nil || target.RawQuery != "" || target.ForceQuery || target.Fragment != "" {
			return fmt.Errorf("mount TCP target cannot contain userinfo, query, or fragment")
		}
		if target.Path != "" && target.Path != "/" {
			return fmt.Errorf("mount TCP target path must be empty or /")
		}
		if err := ValidateConnectAuthority(target.Host); err != nil {
			return fmt.Errorf("mount TCP target: %w", err)
		}
	}
	if strings.Count(target.Path, "*") > 1 || (strings.Contains(target.Path, "*") && !strings.HasSuffix(target.Path, "*")) {
		return fmt.Errorf("mount target %q has an invalid wildcard", m.Target)
	}
	if m.TLS != nil {
		if !strings.EqualFold(target.Scheme, "https") {
			return fmt.Errorf("mount backend TLS policy requires an HTTPS target")
		}
		if err := m.TLS.Validate(); err != nil {
			return fmt.Errorf("mount backend TLS: %w", err)
		}
	}
	return nil
}

// Pattern returns the configured source identity used by policy and
// explanation layers.
func (m Mount) Pattern() string {
	if m.Source != "" {
		return m.Source
	}
	return m.Path
}

func parseSourceURL(source string) (*url.URL, error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("mount source URL: %w", err)
	}
	if !parsed.IsAbs() || parsed.Host == "" || parsed.Opaque != "" {
		return nil, fmt.Errorf("mount source %q requires an absolute authority", source)
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") && !strings.EqualFold(parsed.Scheme, "connect") {
		return nil, fmt.Errorf("mount source scheme %q is unsupported", parsed.Scheme)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("mount source URL cannot contain userinfo")
	}
	if parsed.RawQuery != "" || parsed.ForceQuery {
		return nil, fmt.Errorf("mount source URL cannot contain a query")
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("mount source URL cannot contain a fragment")
	}
	return parsed, nil
}

// ValidateConnectAuthority checks an HTTP CONNECT authority without resolving
// its host or opening a network connection.
func ValidateConnectAuthority(authority string) error {
	if authority == "" || authority != strings.TrimSpace(authority) || strings.ContainsAny(authority, "@/?#") {
		return fmt.Errorf("invalid CONNECT authority %q", authority)
	}
	host, port, err := net.SplitHostPort(authority)
	if err != nil || host == "" || port == "" {
		return fmt.Errorf("CONNECT authority %q requires host and port", authority)
	}
	if strings.ContainsAny(host, " \t\r\n%") {
		return fmt.Errorf("CONNECT authority %q has an invalid host", authority)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 || strconv.Itoa(number) != port {
		return fmt.Errorf("CONNECT authority %q has an invalid port", authority)
	}
	return nil
}
