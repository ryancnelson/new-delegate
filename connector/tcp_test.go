package connector

import (
	"context"
	"net"
	"testing"

	"gitea.local/ryan/new-delegate/operation"
)

func TestTCPValidatesRelayBeforeDialing(t *testing.T) {
	dialer := &recordingDialer{}
	connector := NewTCP(dialer)
	for _, relay := range []operation.Relay{
		{Network: "udp", Address: "example.com:443"},
		{Network: "tcp", Address: "example.com"},
		{Network: "tcp", Address: "example.com:0"},
	} {
		if connection, err := connector.Connect(context.Background(), relay); err == nil || connection != nil {
			t.Errorf("Connect(%#v) = %#v, %v; want validation error", relay, connection, err)
		}
	}
	if dialer.calls != 0 {
		t.Fatalf("dial calls = %d, want zero", dialer.calls)
	}
}

type recordingDialer struct {
	calls int
}

func (d *recordingDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	d.calls++
	return nil, nil
}
