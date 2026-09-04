package explain

import (
	"testing"

	"gitea.local/ryan/new-delegate/config"
	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/policy"
)

func TestEvaluateExplainsPermittedRequest(t *testing.T) {
	configured := config.Config{
		Servers: []config.Server{{Name: "public", Protocol: "http", Listen: ":8080"}},
		Mounts:  []mount.Mount{{Path: "/api/*", Target: "http://api.internal/v1/*"}},
		Policies: []policy.Rule{{
			Effect: policy.Permit, Protocol: "http", Destination: "api.internal",
			Source: "10.0.0.0/8", Method: "GET", Mount: "/api/*",
		}},
	}

	got, err := Evaluate(configured, Request{
		Path: "/api/users", Source: "10.20.30.40", Protocol: "http", Method: "GET",
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.Outcome != OutcomePermit || got.Mount == nil || got.Mount.Target != "http://api.internal/v1/users" {
		t.Fatalf("Evaluate() = %#v, want permitted rewritten target", got)
	}
	if got.Policy == nil || !got.Policy.Allowed || got.Policy.RuleIndex != 0 || got.Policy.Reason != policy.ReasonExplicitPermit {
		t.Fatalf("Evaluate().Policy = %#v, want explicit permit at rule 0", got.Policy)
	}
}

func TestEvaluateExplainsDefaultDeny(t *testing.T) {
	configured := config.Config{
		Servers: []config.Server{{Name: "public", Protocol: "http", Listen: ":8080"}},
		Mounts:  []mount.Mount{{Path: "/*", Target: "http://backend.internal/*"}},
	}
	got, err := Evaluate(configured, Request{Path: "/docs", Source: "192.0.2.4", Protocol: "http", Method: "GET"})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.Outcome != OutcomeReject || got.Policy == nil || got.Policy.Reason != policy.ReasonDefaultDeny {
		t.Fatalf("Evaluate() = %#v, want default deny", got)
	}
}

func TestEvaluateExplainsNoMountAndUnsafePath(t *testing.T) {
	configured := config.Config{
		Servers: []config.Server{{Name: "public", Protocol: "http", Listen: ":8080"}},
		Mounts:  []mount.Mount{{Path: "/api/*", Target: "http://api.internal/*"}},
	}
	tests := []struct {
		path    string
		outcome Outcome
	}{
		{path: "/docs", outcome: OutcomeNoMount},
		{path: "/../secret", outcome: OutcomeUnsafePath},
	}
	for _, tt := range tests {
		t.Run(string(tt.outcome), func(t *testing.T) {
			got, err := Evaluate(configured, Request{Path: tt.path, Source: "192.0.2.4", Protocol: "http", Method: "GET"})
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got.Outcome != tt.outcome || got.Policy != nil {
				t.Fatalf("Evaluate() = %#v, want outcome %q without policy decision", got, tt.outcome)
			}
		})
	}
}

func TestEvaluateExplainsAmbiguousMount(t *testing.T) {
	configured := config.Config{
		Servers: []config.Server{{Name: "public", Protocol: "http", Listen: ":8080"}},
		Mounts: []mount.Mount{
			{Path: "/api/*", Target: "http://one.internal/*", Priority: 10},
			{Path: "/api/*", Target: "http://two.internal/*", Priority: 10},
		},
	}
	got, err := Evaluate(configured, Request{Path: "/api/users", Source: "192.0.2.4", Protocol: "http", Method: "GET"})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.Outcome != OutcomeAmbiguous || got.Mount != nil || got.Policy != nil {
		t.Fatalf("Evaluate() = %#v, want ambiguous mount without policy evaluation", got)
	}
}

func TestEvaluateRejectsIncompleteRequest(t *testing.T) {
	configured := config.Config{Servers: []config.Server{{Name: "public", Protocol: "http", Listen: ":8080"}}}
	if _, err := Evaluate(configured, Request{}); err == nil {
		t.Fatal("Evaluate() error = nil, want incomplete request error")
	}
}
