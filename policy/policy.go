// Package policy makes explicit, fail-closed authorization decisions.
package policy

import (
	"errors"
	"fmt"
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
	Source   string
	Protocol string
	Method   string
	Mount    string
}

// Rule matches non-empty constraints conjunctively. Empty and "*" match any
// value. Higher priority wins; Reject wins an equal-priority tie.
type Rule struct {
	Effect   Effect
	Priority int
	Source   string
	Protocol string
	Method   string
	Mount    string
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

func ruleMatches(rule Rule, request Request) bool {
	return matches(rule.Source, request.Source, false) &&
		matches(rule.Protocol, request.Protocol, true) &&
		matches(rule.Method, request.Method, true) &&
		matches(rule.Mount, request.Mount, false)
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
