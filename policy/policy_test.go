package policy

import (
	"errors"
	"testing"
)

func TestEvaluate(t *testing.T) {
	request := Request{Source: "127.0.0.1", Protocol: "http", Method: "GET", Mount: "/api/*"}
	tests := []struct {
		name        string
		rules       []Rule
		wantAllowed bool
		wantCode    ReasonCode
	}{
		{name: "default deny", wantCode: ReasonDefaultDeny},
		{
			name:        "explicit permit",
			rules:       []Rule{{Effect: Permit, Protocol: "http", Method: "GET", Priority: 10}},
			wantAllowed: true,
			wantCode:    ReasonExplicitPermit,
		},
		{
			name:     "explicit reject",
			rules:    []Rule{{Effect: Reject, Protocol: "http", Method: "GET", Priority: 10}},
			wantCode: ReasonExplicitReject,
		},
		{
			name: "reject wins at equal priority",
			rules: []Rule{
				{Effect: Permit, Protocol: "http", Method: "GET", Priority: 20},
				{Effect: Reject, Source: "127.0.0.1", Priority: 20},
			},
			wantCode: ReasonExplicitReject,
		},
		{
			name: "higher priority permit wins",
			rules: []Rule{
				{Effect: Reject, Protocol: "http", Priority: 10},
				{Effect: Permit, Protocol: "http", Method: "GET", Mount: "/api/*", Priority: 20},
			},
			wantAllowed: true,
			wantCode:    ReasonExplicitPermit,
		},
		{
			name: "nonmatching permit still denies",
			rules: []Rule{
				{Effect: Permit, Protocol: "ftp", Priority: 100},
			},
			wantCode: ReasonDefaultDeny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.rules, request)
			if got.Allowed != tt.wantAllowed || got.Reason != tt.wantCode {
				t.Fatalf("Evaluate() = %#v, want allowed=%v reason=%q", got, tt.wantAllowed, tt.wantCode)
			}
		})
	}
}

func TestDecisionEnforceNeverInvokesDeniedOperation(t *testing.T) {
	called := false
	decision := Evaluate(nil, Request{Protocol: "http", Method: "GET"})
	err := decision.Enforce(func() error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Enforce() error = %v, want ErrDenied", err)
	}
	if called {
		t.Fatal("denied operation invoked connector callback")
	}
}

func TestDecisionEnforceInvokesPermittedOperation(t *testing.T) {
	called := false
	decision := Evaluate(
		[]Rule{{Effect: Permit, Protocol: "http"}},
		Request{Protocol: "http", Method: "GET"},
	)
	err := decision.Enforce(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("Enforce() error = %v, want nil", err)
	}
	if !called {
		t.Fatal("permitted operation did not invoke connector callback")
	}
}

func TestEvaluateLegacySelectors(t *testing.T) {
	rules := []Rule{
		{Effect: Permit, Protocol: "http", Destination: "*.internal.com", Source: "10.0.0.0/8", Priority: 0},
		{Effect: Reject, Protocol: "*", Destination: "*", Source: "*", Priority: -1},
	}
	tests := []struct {
		name    string
		request Request
		allowed bool
	}{
		{
			name:    "CIDR and destination suffix match",
			request: Request{Protocol: "HTTP", Destination: "api.internal.com", Source: "10.20.30.40"},
			allowed: true,
		},
		{
			name:    "suffix does not match bare parent",
			request: Request{Protocol: "http", Destination: "internal.com", Source: "10.20.30.40"},
		},
		{
			name:    "CIDR excludes source",
			request: Request{Protocol: "http", Destination: "api.internal.com", Source: "192.0.2.4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Evaluate(rules, tt.request); got.Allowed != tt.allowed {
				t.Fatalf("Evaluate() = %#v, want allowed=%v", got, tt.allowed)
			}
		})
	}
}
