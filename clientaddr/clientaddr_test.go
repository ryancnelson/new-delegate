package clientaddr

import (
	"net/netip"
	"testing"
)

func TestResolveIgnoresForwardingFromUntrustedPeer(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	got, err := Resolve("192.0.2.10:1234", "203.0.113.7", trusted)
	if err != nil || got != "192.0.2.10" {
		t.Fatalf("Resolve() = %q, %v; want direct peer", got, err)
	}
}

func TestResolveWalksTrustedChainFromRight(t *testing.T) {
	trusted := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("192.168.0.0/16"),
	}
	got, err := Resolve("10.0.0.2:1234", "203.0.113.7, 192.168.1.9", trusted)
	if err != nil || got != "203.0.113.7" {
		t.Fatalf("Resolve() = %q, %v; want original untrusted client", got, err)
	}
}

func TestResolveRejectsMalformedTrustedChain(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	if _, err := Resolve("10.0.0.2:1234", "not-an-address", trusted); err == nil {
		t.Fatal("Resolve() error = nil, want malformed chain error")
	}
}

func TestResolveUsesDirectPeerWithoutHeader(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	got, err := Resolve("10.0.0.2:1234", "", trusted)
	if err != nil || got != "10.0.0.2" {
		t.Fatalf("Resolve() = %q, %v; want direct peer", got, err)
	}
}

func TestResolveMatchesIPv4MappedTrustedPeer(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	got, err := Resolve("[::ffff:10.0.0.2]:1234", "203.0.113.7", trusted)
	if err != nil || got != "203.0.113.7" {
		t.Fatalf("Resolve() = %q, %v; want forwarded client", got, err)
	}
}
