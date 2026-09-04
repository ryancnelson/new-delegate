package config

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/policy"
	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestStoreReplacesWholeValidatedSnapshot(t *testing.T) {
	initial := storeTestConfig("old.internal", 8080)
	store, err := NewStore(initial)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	candidate := storeTestConfig("new.internal", 8081)
	if err := store.Replace(candidate); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	candidate.Servers[0].Name = "mutated-after-replace"
	candidate.Mounts[0].Target = "http://mutated.invalid/*"

	want := storeTestConfig("new.internal", 8081)
	if got := store.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Snapshot() = %#v, want %#v", got, want)
	}
}

func TestStoreRejectsInvalidReplacementWithoutChangingSnapshot(t *testing.T) {
	initial := storeTestConfig("old.internal", 8080)
	store, err := NewStore(initial)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	invalid := storeTestConfig("broken.internal", 8081)
	invalid.Mounts[0].Path = "relative"
	if err := store.Replace(invalid); err == nil {
		t.Fatal("Replace() error = nil, want validation failure")
	}
	if got := store.Snapshot(); !reflect.DeepEqual(got, initial) {
		t.Fatalf("Snapshot() changed after rejected replacement: %#v", got)
	}
}

func TestStoreSnapshotsDoNotExposeMutableState(t *testing.T) {
	initial := storeTestConfig("old.internal", 8080)
	initial.Servers[0].ClientIPHeader = "X-Forwarded-For"
	initial.Servers[0].TrustedProxies = []string{"10.0.0.0/8"}
	initial.Servers[0].TLS = &tlsconfig.Frontend{
		CertificateFile: "cert.pem", PrivateKeyFile: "key.pem",
	}
	initial.Mounts[0].Target = "https://old.internal/*"
	initial.Mounts[0].TLS = &tlsconfig.Backend{CAFile: "ca.pem"}
	store, err := NewStore(initial)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	snapshot := store.Snapshot()
	snapshot.Servers[0].Name = "mutated"
	snapshot.Servers[0].TrustedProxies[0] = "192.0.2.0/24"
	snapshot.Servers[0].TLS.CertificateFile = "mutated.pem"
	snapshot.Mounts[0].Target = "http://mutated.invalid/*"
	snapshot.Mounts[0].TLS.CAFile = "mutated-ca.pem"
	snapshot.Policies[0].Effect = policy.Reject
	if got := store.Snapshot(); !reflect.DeepEqual(got, initial) {
		t.Fatalf("Snapshot() exposed stored state: %#v", got)
	}
}

func TestNewStoreRejectsInvalidInitialConfiguration(t *testing.T) {
	if _, err := NewStore(Config{}); err == nil {
		t.Fatal("NewStore() error = nil, want validation failure")
	}
}

func TestStoreConcurrentSnapshotsAndReplacements(t *testing.T) {
	store, err := NewStore(storeTestConfig("initial.internal", 8080))
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for reader := 0; reader < 8; reader++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 500 {
				snapshot := store.Snapshot()
				if err := snapshot.Validate(); err != nil {
					t.Errorf("observed invalid snapshot: %v", err)
					return
				}
			}
		}()
	}
	for replacement := 0; replacement < 100; replacement++ {
		if err := store.Replace(storeTestConfig(fmt.Sprintf("backend-%d.internal", replacement), 8080+replacement%2)); err != nil {
			t.Fatal(err)
		}
	}
	group.Wait()
}

func storeTestConfig(host string, port int) Config {
	return Config{
		Servers: []Server{{Name: "public", Protocol: "http", Listen: fmt.Sprintf(":%d", port)}},
		Mounts:  []mount.Mount{{Path: "/*", Target: "http://" + host + "/*"}},
		Policies: []policy.Rule{{
			Effect: policy.Permit, Protocol: "http", Destination: host, Source: "*",
		}},
	}
}
