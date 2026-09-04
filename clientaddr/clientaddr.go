// Package clientaddr resolves a policy source address across explicitly
// trusted reverse proxies.
package clientaddr

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
)

// Resolve returns the direct peer unless that peer is trusted and supplied a
// forwarding chain. Trusted chains are walked from the nearest hop toward the
// client; the first untrusted address is authoritative.
func Resolve(remoteAddress, forwarded string, trusted []netip.Prefix) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		host = remoteAddress
	}
	peer, err := netip.ParseAddr(strings.TrimSpace(host))
	if err != nil {
		return "", fmt.Errorf("invalid direct peer address %q", remoteAddress)
	}
	peer = peer.Unmap()
	if !contains(trusted, peer) || strings.TrimSpace(forwarded) == "" {
		return peer.String(), nil
	}

	parts := strings.Split(forwarded, ",")
	addresses := make([]netip.Addr, len(parts))
	for i, part := range parts {
		address, err := netip.ParseAddr(strings.TrimSpace(part))
		if err != nil {
			return "", fmt.Errorf("invalid forwarded address %q", strings.TrimSpace(part))
		}
		addresses[i] = address.Unmap()
	}
	for i := len(addresses) - 1; i >= 0; i-- {
		if !contains(trusted, addresses[i]) {
			return addresses[i].String(), nil
		}
	}
	return addresses[0].String(), nil
}

func contains(prefixes []netip.Prefix, address netip.Addr) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
