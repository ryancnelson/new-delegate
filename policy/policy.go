// Package policy makes explicit, fail-closed authorization decisions.
package policy

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

type Effect string

const (
	Permit Effect = "permit"
	Reject Effect = "reject"
)

type ReasonCode string

const (
	ReasonDefaultDeny    ReasonCode = "default_deny"
	ReasonExplicitPermit ReasonCode = "explicit_permit"
	ReasonExplicitReject ReasonCode = "explicit_reject"
)

var ErrDenied = errors.New("operation denied")

// Request is the protocol-neutral authorization input.
type Request struct {
	Source      string
	Protocol    string
	Destination string
	Method      string
	Mount       string
}

// Rule matches non-empty constraints conjunctively. Empty and "*" match any
// value. Higher priority wins; Reject wins an equal-priority tie.
type Rule struct {
	Effect      Effect `json:"effect" toml:"effect"`
	Priority    int    `json:"priority" toml:"priority"`
	Source      string `json:"source,omitempty" toml:"source"`
	Protocol    string `json:"protocol,omitempty" toml:"protocol"`
	Destination string `json:"destination,omitempty" toml:"destination"`
	Method      string `json:"method,omitempty" toml:"method"`
	Mount       string `json:"mount,omitempty" toml:"mount"`
}

// Decision is a typed, auditable policy result.
type Decision struct {
	Allowed   bool
	Reason    ReasonCode
	RuleIndex int
}

// Evaluate selects the highest-priority matching rule. Absence of an explicit
// permit is always a denial.
func Evaluate(rules []Rule, request Request) Decision {
	winner := -1
	for i, rule := range rules {
		if (rule.Effect != Permit && rule.Effect != Reject) || !ruleMatches(rule, request) {
			continue
		}
		if winner == -1 || rule.Priority > rules[winner].Priority ||
			(rule.Priority == rules[winner].Priority && rule.Effect == Reject && rules[winner].Effect != Reject) {
			winner = i
		}
	}
	if winner == -1 {
		return Decision{Reason: ReasonDefaultDeny, RuleIndex: -1}
	}
	if rules[winner].Effect == Reject {
		return Decision{Reason: ReasonExplicitReject, RuleIndex: winner}
	}
	return Decision{Allowed: true, Reason: ReasonExplicitPermit, RuleIndex: winner}
}

// Enforce invokes next only for an affirmative decision.
func (d Decision) Enforce(next func() error) error {
	if !d.Allowed {
		return fmt.Errorf("%w: %s", ErrDenied, d.Reason)
	}
	if next == nil {
		return nil
	}
	return next()
}

// Validate checks selector syntax without resolving names or performing I/O.
func (r Rule) Validate() error {
	if r.Effect != Permit && r.Effect != Reject {
		return fmt.Errorf("unknown policy effect %q", r.Effect)
	}
	if r.Source != "" && r.Source != "*" {
		if strings.Contains(r.Source, "/") {
			if _, err := netip.ParsePrefix(r.Source); err != nil {
				return fmt.Errorf("invalid source CIDR %q", r.Source)
			}
		} else if _, err := netip.ParseAddr(r.Source); err != nil {
			return fmt.Errorf("invalid source address %q", r.Source)
		}
	}
	if strings.Contains(r.Destination, "*") && r.Destination != "*" &&
		(!strings.HasPrefix(r.Destination, "*.") || strings.Count(r.Destination, "*") != 1) {
		return fmt.Errorf("invalid destination wildcard %q", r.Destination)
	}
	return nil
}

func ruleMatches(rule Rule, request Request) bool {
	return sourceMatches(rule.Source, request.Source) &&
		matches(rule.Protocol, request.Protocol, true) &&
		destinationMatches(rule.Destination, request.Destination) &&
		matches(rule.Method, request.Method, true) &&
		matches(rule.Mount, request.Mount, false)
}

func sourceMatches(pattern, value string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	if prefix, err := netip.ParsePrefix(pattern); err == nil {
		address, err := netip.ParseAddr(value)
		return err == nil && prefix.Contains(address)
	}
	return pattern == value
}

func destinationMatches(pattern, value string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.ToLower(strings.TrimPrefix(pattern, "*"))
		return strings.HasSuffix(strings.ToLower(value), suffix) && len(value) > len(suffix)
	}
	return strings.EqualFold(pattern, value)
}

func matches(pattern, value string, foldCase bool) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	if foldCase {
		return strings.EqualFold(pattern, value)
	}
	return pattern == value
}
