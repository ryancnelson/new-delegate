package config

import (
	"fmt"
	"strconv"
	"strings"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/policy"
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
	var policies []policy.Rule

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
		case strings.HasPrefix(arg, "PERMIT="):
			rule, err := parseLegacyPolicy(policy.Permit, strings.TrimSpace(strings.TrimPrefix(arg, "PERMIT=")), -len(policies))
			if err != nil {
				return Config{}, err
			}
			policies = append(policies, rule)
		case strings.HasPrefix(arg, "REJECT="):
			rule, err := parseLegacyPolicy(policy.Reject, strings.TrimSpace(strings.TrimPrefix(arg, "REJECT=")), -len(policies))
			if err != nil {
				return Config{}, err
			}
			policies = append(policies, rule)
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
	}}, Mounts: mounts, Policies: policies}
	if err := result.Validate(); err != nil {
		return Config{}, err
	}
	return result, nil
}

func parseLegacyPolicy(effect policy.Effect, value string, priority int) (policy.Rule, error) {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return policy.Rule{}, fmt.Errorf("invalid %s quoting: %w", strings.ToUpper(string(effect)), err)
		}
		value = unquoted
	}
	selectors := strings.SplitN(value, ":", 3)
	if len(selectors) != 3 {
		return policy.Rule{}, fmt.Errorf("%s requires three selectors: protocol:destination:source", strings.ToUpper(string(effect)))
	}
	for i := range selectors {
		selectors[i] = strings.TrimSpace(selectors[i])
		if selectors[i] == "" {
			return policy.Rule{}, fmt.Errorf("%s contains an empty selector", strings.ToUpper(string(effect)))
		}
	}
	rule := policy.Rule{
		Effect:      effect,
		Priority:    priority,
		Protocol:    strings.ToLower(selectors[0]),
		Destination: strings.ToLower(selectors[1]),
		Source:      selectors[2],
	}
	if err := rule.Validate(); err != nil {
		return policy.Rule{}, fmt.Errorf("invalid %s: %w", strings.ToUpper(string(effect)), err)
	}
	return rule, nil
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
	if strings.Contains(fields[0], "://") {
		mapping.Source = fields[0]
		mapping.Path = ""
	}
	for _, option := range fields[2:] {
		key, value, hasValue := strings.Cut(option, "=")
		if !hasValue {
			return mount.Mount{}, fmt.Errorf("unknown MOUNT option %q", option)
		}
		switch strings.ToLower(key) {
		case "priority":
			priority, err := strconv.Atoi(value)
			if err != nil {
				return mount.Mount{}, fmt.Errorf("invalid MOUNT priority %q", option)
			}
			mapping.Priority = priority
		case "server":
			mapping.Server = value
		case "protocol":
			mapping.Protocol = strings.ToLower(value)
		default:
			return mount.Mount{}, fmt.Errorf("unknown MOUNT option %q", option)
		}
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
