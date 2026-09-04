// Package explain evaluates routing and policy without opening runtime
// resources or invoking a backend connector.
package explain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"gitea.local/ryan/new-delegate/config"
	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/policy"
)

// Outcome is the stable top-level result of an explanation.
type Outcome string

const (
	OutcomePermit     Outcome = "permit"
	OutcomeReject     Outcome = "reject"
	OutcomeNoMount    Outcome = "no_mount"
	OutcomeUnsafePath Outcome = "unsafe_path"
	OutcomeAmbiguous  Outcome = "ambiguous_mount"
)

// Request is the complete protocol-neutral input to a routing explanation.
type Request struct {
	Path     string `json:"path"`
	URL      string `json:"url,omitempty"`
	Source   string `json:"source"`
	Server   string `json:"server"`
	Protocol string `json:"protocol"`
	Method   string `json:"method"`
}

// MountResult describes the winning mount and rewritten backend target.
type MountResult struct {
	Path           string `json:"path,omitempty"`
	Source         string `json:"source,omitempty"`
	NormalizedPath string `json:"normalized_path"`
	Target         string `json:"target"`
}

// PolicyResult describes the effective policy decision and winning rule.
type PolicyResult struct {
	Allowed   bool              `json:"allowed"`
	Reason    policy.ReasonCode `json:"reason"`
	RuleIndex int               `json:"rule_index"`
}

// Result is a side-effect-free explanation of mount and policy evaluation.
type Result struct {
	Outcome Outcome       `json:"outcome"`
	Request Request       `json:"request"`
	Mount   *MountResult  `json:"mount,omitempty"`
	Policy  *PolicyResult `json:"policy,omitempty"`
}

// Evaluate runs the same mount and policy kernels used by the HTTP frontend.
func Evaluate(configured config.Config, request Request) (Result, error) {
	if err := configured.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate configuration: %w", err)
	}
	if (strings.TrimSpace(request.Path) == "") == (strings.TrimSpace(request.URL) == "") {
		return Result{}, fmt.Errorf("request requires exactly one path or absolute URL")
	}
	if strings.TrimSpace(request.Source) == "" {
		return Result{}, fmt.Errorf("request source is required")
	}
	if strings.TrimSpace(request.Protocol) == "" {
		return Result{}, fmt.Errorf("request protocol is required")
	}
	if strings.TrimSpace(request.Server) == "" {
		return Result{}, fmt.Errorf("request server is required")
	}
	if strings.TrimSpace(request.Method) == "" {
		return Result{}, fmt.Errorf("request method is required")
	}

	mountRequest := mount.Request{
		Path: request.Path, Server: request.Server, Protocol: request.Protocol,
	}
	if request.URL != "" {
		parsed, err := url.ParseRequestURI(request.URL)
		if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
			return Result{}, fmt.Errorf("request URL must be an absolute URL without userinfo or fragment")
		}
		mountRequest.Path = parsed.EscapedPath()
		if mountRequest.Path == "" {
			mountRequest.Path = "/"
		}
		mountRequest.Scheme = parsed.Scheme
		mountRequest.Authority = parsed.Host
		request.Path = mountRequest.Path
	}
	result := Result{Request: request}
	matched, err := mount.ResolveFor(configured.Mounts, mount.Request{
		Path: mountRequest.Path, Scheme: mountRequest.Scheme, Authority: mountRequest.Authority,
		Server: mountRequest.Server, Protocol: mountRequest.Protocol,
	})
	if err != nil {
		switch {
		case errors.Is(err, mount.ErrNoMatch):
			result.Outcome = OutcomeNoMount
		case errors.Is(err, mount.ErrUnsafePath):
			result.Outcome = OutcomeUnsafePath
		case errors.Is(err, mount.ErrAmbiguous):
			result.Outcome = OutcomeAmbiguous
		default:
			return Result{}, fmt.Errorf("resolve mount: %w", err)
		}
		return result, nil
	}

	result.Mount = &MountResult{
		Path:           matched.Mount.Path,
		Source:         matched.Mount.Source,
		NormalizedPath: matched.NormalizedPath,
		Target:         matched.Target,
	}
	target, err := url.Parse(matched.Target)
	if err != nil {
		return Result{}, fmt.Errorf("parse matched target: %w", err)
	}
	decision := policy.Evaluate(configured.Policies, policy.Request{
		Source:      request.Source,
		Protocol:    request.Protocol,
		Destination: target.Hostname(),
		Method:      request.Method,
		Mount:       matched.Mount.Pattern(),
	})
	result.Policy = &PolicyResult{
		Allowed: decision.Allowed, Reason: decision.Reason, RuleIndex: decision.RuleIndex,
	}
	if decision.Allowed {
		result.Outcome = OutcomePermit
	} else {
		result.Outcome = OutcomeReject
	}
	return result, nil
}
