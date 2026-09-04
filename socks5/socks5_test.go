package socks5

import (
	"bytes"
	"errors"
	"testing"
)

func TestReadGreetingAndSelectNoAuth(t *testing.T) {
	t.Parallel()

	greeting, err := ReadGreeting(bytes.NewReader([]byte{Version, 2, 0x02, MethodNoAuth}))
	if err != nil {
		t.Fatalf("ReadGreeting() = %v", err)
	}
	if !greeting.Offers(MethodNoAuth) {
		t.Fatal("Greeting.Offers(MethodNoAuth) = false, want true")
	}
	if got := SelectNoAuth(greeting); got != MethodNoAuth {
		t.Fatalf("SelectNoAuth() = %#x, want %#x", got, MethodNoAuth)
	}

	var encoded bytes.Buffer
	if err := WriteMethod(&encoded, MethodNoAuth); err != nil {
		t.Fatalf("WriteMethod() = %v", err)
	}
	if got, want := encoded.Bytes(), []byte{Version, MethodNoAuth}; !bytes.Equal(got, want) {
		t.Fatalf("WriteMethod() = %x, want %x", got, want)
	}
}

func TestSelectNoAuthRejectsUnsupportedGreeting(t *testing.T) {
	t.Parallel()

	greeting := Greeting{Methods: []byte{0x02}}
	if got := SelectNoAuth(greeting); got != MethodNoAcceptable {
		t.Fatalf("SelectNoAuth() = %#x, want %#x", got, MethodNoAcceptable)
	}
}

func TestReadGreetingRejectsMalformedFrames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		wire []byte
		want error
	}{
		{name: "truncated", wire: []byte{Version}, want: nil},
		{name: "wrong version", wire: []byte{4, 1, MethodNoAuth}, want: ErrUnsupportedVersion},
		{name: "no methods", wire: []byte{Version, 0}, want: ErrNoMethods},
		{name: "truncated method list", wire: []byte{Version, 2, MethodNoAuth}, want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ReadGreeting(bytes.NewReader(test.wire))
			if err == nil {
				t.Fatal("ReadGreeting() = nil, want error")
			}
			if test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("ReadGreeting() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestReadRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		wire []byte
		want Request
	}{
		{
			name: "ipv4",
			wire: []byte{Version, CommandConnect, 0, AddressIPv4, 192, 0, 2, 9, 0x01, 0xbb},
			want: Request{Command: CommandConnect, Host: "192.0.2.9", Port: 443},
		},
		{
			name: "domain",
			wire: append([]byte{Version, CommandConnect, 0, AddressDomain, 11}, append([]byte("example.com"), 0, 80)...),
			want: Request{Command: CommandConnect, Host: "example.com", Port: 80},
		},
		{
			name: "ipv6",
			wire: []byte{Version, CommandConnect, 0, AddressIPv6, 0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 22},
			want: Request{Command: CommandConnect, Host: "2001:db8::1", Port: 22},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ReadRequest(bytes.NewReader(test.wire))
			if err != nil {
				t.Fatalf("ReadRequest() = %v", err)
			}
			if got != test.want {
				t.Fatalf("ReadRequest() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestReadRequestRejectsMalformedFrames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		wire []byte
	}{
		{name: "truncated header", wire: []byte{Version, CommandConnect}},
		{name: "wrong version", wire: []byte{4, CommandConnect, 0, AddressIPv4, 192, 0, 2, 1, 0, 80}},
		{name: "unsupported command", wire: []byte{Version, 2, 0, AddressIPv4, 192, 0, 2, 1, 0, 80}},
		{name: "reserved byte", wire: []byte{Version, CommandConnect, 1, AddressIPv4, 192, 0, 2, 1, 0, 80}},
		{name: "unsupported address type", wire: []byte{Version, CommandConnect, 0, 2}},
		{name: "truncated address", wire: []byte{Version, CommandConnect, 0, AddressIPv4}},
		{name: "empty domain", wire: []byte{Version, CommandConnect, 0, AddressDomain, 0, 0, 80}},
		{name: "invalid domain", wire: []byte{Version, CommandConnect, 0, AddressDomain, 3, 'a', ' ', 'b', 0, 80}},
		{name: "zero port", wire: []byte{Version, CommandConnect, 0, AddressIPv4, 192, 0, 2, 1, 0, 0}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ReadRequest(bytes.NewReader(test.wire)); err == nil {
				t.Fatal("ReadRequest() = nil, want error")
			}
		})
	}
}

func TestWriteReply(t *testing.T) {
	t.Parallel()

	var encoded bytes.Buffer
	if err := WriteReply(&encoded, ReplyCommandNotSupported); err != nil {
		t.Fatalf("WriteReply() = %v", err)
	}
	want := []byte{Version, ReplyCommandNotSupported, 0, AddressIPv4, 0, 0, 0, 0, 0, 0}
	if got := encoded.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("WriteReply() = %x, want %x", got, want)
	}
}

func TestWriteRejectsInvalidReplyOrMethod(t *testing.T) {
	t.Parallel()

	if err := WriteMethod(&bytes.Buffer{}, 0x7f); !errors.Is(err, ErrUnsupportedMethod) {
		t.Fatalf("WriteMethod() error = %v, want ErrUnsupportedMethod", err)
	}
	if err := WriteReply(&bytes.Buffer{}, 0x7f); !errors.Is(err, ErrUnsupportedReply) {
		t.Fatalf("WriteReply() error = %v, want ErrUnsupportedReply", err)
	}
}

func FuzzReadGreeting(f *testing.F) {
	f.Add([]byte{Version, 1, MethodNoAuth})
	f.Add([]byte{Version, 0})
	f.Fuzz(func(t *testing.T, wire []byte) {
		_, _ = ReadGreeting(bytes.NewReader(wire))
	})
}

func FuzzReadRequest(f *testing.F) {
	f.Add([]byte{Version, CommandConnect, 0, AddressIPv4, 192, 0, 2, 1, 0, 80})
	f.Add([]byte{Version, CommandConnect, 0, AddressDomain, 11, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 0, 80})
	f.Fuzz(func(t *testing.T, wire []byte) {
		_, _ = ReadRequest(bytes.NewReader(wire))
	})
}
