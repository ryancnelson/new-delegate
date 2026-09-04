package connector

import (
	"context"
	"fmt"
	"net"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/operation"
)

type contextDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

// TCP opens transparent relay streams through an injected bounded dialer.
type TCP struct {
	dialer contextDialer
}

// NewTCP constructs a TCP relay connector.
func NewTCP(dialer contextDialer) *TCP {
	return &TCP{dialer: dialer}
}

// Connect opens the requested stream after the frontend has authorized it.
func (t *TCP) Connect(ctx context.Context, relay operation.Relay) (net.Conn, error) {
	if t == nil || t.dialer == nil {
		return nil, fmt.Errorf("TCP dialer is required")
	}
	if relay.Network != "tcp" {
		return nil, fmt.Errorf("unsupported relay network %q", relay.Network)
	}
	if err := mount.ValidateConnectAuthority(relay.Address); err != nil {
		return nil, err
	}
	return t.dialer.DialContext(ctx, relay.Network, relay.Address)
}
