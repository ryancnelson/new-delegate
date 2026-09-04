package config

import (
	"fmt"
	"strconv"
	"strings"

	"gitea.local/ryan/new-delegate/mount"
)

var legacyDefaultPorts = map[string]int{
	"ftp":    21,
	"gopher": 70,
	"http":   80,
	"https":  443,
	"socks":  1080,
}

// ParseLegacyArgs converts original DeleGate-style command-line directives
// into canonical configuration. It only parses data; it never starts a
// listener or performs network I/O.
func ParseLegacyArgs(args []string) (Config, error) {
	var protocol string
	var port int
	var mounts []mount.Mount

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case strings.HasPrefix(arg, "SERVER="):
			value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "SERVER=")))
			if value == "" {
				return Config{}, fmt.Errorf("SERVER protocol is required")
			}
			if protocol != "" && protocol != value {
				return Config{}, fmt.Errorf("conflicting SERVER directives %q and %q", protocol, value)
			}
			protocol = value
		case arg == "-P":
			i++
			if i >= len(args) {
				return Config{}, fmt.Errorf("invalid port: -P requires a value")
			}
			value, err := parseLegacyPort(strings.TrimSpace(args[i]))
			if err != nil {
				return Config{}, err
			}
			if port != 0 && port != value {
				return Config{}, fmt.Errorf("conflicting -P directives %d and %d", port, value)
			}
			port = value
		case strings.HasPrefix(arg, "-P"):
			value, err := parseLegacyPort(strings.TrimSpace(strings.TrimPrefix(arg, "-P")))
			if err != nil {
				return Config{}, err
			}
			if port != 0 && port != value {
				return Config{}, fmt.Errorf("conflicting -P directives %d and %d", port, value)
			}
			port = value
		case strings.HasPrefix(arg, "MOUNT="):
			mapping, err := parseLegacyMount(strings.TrimSpace(strings.TrimPrefix(arg, "MOUNT=")))
			if err != nil {
				return Config{}, err
			}
			mounts = append(mounts, mapping)
		default:
			return Config{}, fmt.Errorf("unknown directive %q", arg)
		}
	}

	if protocol == "" {
		return Config{}, fmt.Errorf("SERVER directive is required")
	}
	if port == 0 {
		var ok bool
		port, ok = legacyDefaultPorts[protocol]
		if !ok {
			return Config{}, fmt.Errorf("SERVER=%s has no default port; specify -P", protocol)
		}
	}

	result := Config{Servers: []Server{{
		Name:     "default",
		Protocol: protocol,
		Listen:   fmt.Sprintf(":%d", port),
	}}, Mounts: mounts}
	if err := result.Validate(); err != nil {
		return Config{}, err
	}
	return result, nil
}

func parseLegacyMount(value string) (mount.Mount, error) {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return mount.Mount{}, fmt.Errorf("invalid MOUNT quoting: %w", err)
		}
		value = unquoted
	}
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return mount.Mount{}, fmt.Errorf("MOUNT requires a path and target")
	}

	mapping := mount.Mount{Path: fields[0], Target: fields[1]}
	for _, option := range fields[2:] {
		if strings.HasPrefix(option, "priority=") {
			priority, err := strconv.Atoi(strings.TrimPrefix(option, "priority="))
			if err != nil {
				return mount.Mount{}, fmt.Errorf("invalid MOUNT priority %q", option)
			}
			mapping.Priority = priority
			continue
		}
		return mount.Mount{}, fmt.Errorf("unknown MOUNT option %q", option)
	}
	if err := mapping.Validate(); err != nil {
		return mount.Mount{}, fmt.Errorf("invalid MOUNT: %w", err)
	}
	return mapping, nil
}

func parseLegacyPort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port %q", value)
	}
	return port, nil
}
