package config

import (
	"fmt"
	"strconv"
	"strings"
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
	}}}
	if err := result.Validate(); err != nil {
		return Config{}, err
	}
	return result, nil
}

func parseLegacyPort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port %q", value)
	}
	return port, nil
}
