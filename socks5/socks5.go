// Package socks5 provides bounded SOCKS5 wire framing for the frontend.
package socks5

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/netip"
)

const (
	// Version is the SOCKS5 protocol version.
	Version byte = 0x05

	// MethodNoAuth is the only authentication method currently supported.
	MethodNoAuth       byte = 0x00
	MethodNoAcceptable byte = 0xff

	// CommandConnect establishes a TCP relay after policy authorization.
	CommandConnect byte = 0x01

	// Supported SOCKS5 destination address encodings.
	AddressIPv4   byte = 0x01
	AddressDomain byte = 0x03
	AddressIPv6   byte = 0x04

	ReplySucceeded            byte = 0x00
	ReplyGeneralFailure       byte = 0x01
	ReplyConnectionNotAllowed byte = 0x02
	ReplyNetworkUnreachable   byte = 0x03
	ReplyHostUnreachable      byte = 0x04
	ReplyConnectionRefused    byte = 0x05
	ReplyTTLExpired           byte = 0x06
	ReplyCommandNotSupported  byte = 0x07
	ReplyAddressNotSupported  byte = 0x08
)

var (
	ErrUnsupportedVersion     = errors.New("unsupported SOCKS version")
	ErrNoMethods              = errors.New("SOCKS greeting offers no methods")
	ErrUnsupportedMethod      = errors.New("unsupported SOCKS authentication method")
	ErrUnsupportedCommand     = errors.New("unsupported SOCKS command")
	ErrInvalidReservedByte    = errors.New("invalid SOCKS reserved byte")
	ErrUnsupportedAddressType = errors.New("unsupported SOCKS address type")
	ErrEmptyDomain            = errors.New("empty SOCKS domain")
	ErrInvalidDomain          = errors.New("invalid SOCKS domain")
	ErrInvalidPort            = errors.New("invalid SOCKS port")
	ErrUnsupportedReply       = errors.New("unsupported SOCKS reply")
)

// Greeting is the method-negotiation offer sent by a SOCKS5 client.
type Greeting struct {
	Methods []byte
}

// Offers reports whether the client offered method.
func (g Greeting) Offers(method byte) bool {
	for _, offered := range g.Methods {
		if offered == method {
			return true
		}
	}
	return false
}

// Request is a parsed SOCKS5 request. Host is a canonical IP address or the
// raw, validated ASCII domain requested by the client. Parsing never resolves
// a domain or opens a connection.
type Request struct {
	Command byte
	Host    string
	Port    uint16
}

// ReadGreeting decodes a SOCKS5 method-negotiation offer.
func ReadGreeting(reader io.Reader) (Greeting, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return Greeting{}, fmt.Errorf("read SOCKS greeting header: %w", err)
	}
	if header[0] != Version {
		return Greeting{}, fmt.Errorf("%w: %#x", ErrUnsupportedVersion, header[0])
	}
	if header[1] == 0 {
		return Greeting{}, ErrNoMethods
	}

	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(reader, methods); err != nil {
		return Greeting{}, fmt.Errorf("read SOCKS greeting methods: %w", err)
	}
	return Greeting{Methods: methods}, nil
}

// SelectNoAuth selects no-authentication when the client offers it, otherwise
// returning the protocol's no-acceptable-methods marker.
func SelectNoAuth(greeting Greeting) byte {
	if greeting.Offers(MethodNoAuth) {
		return MethodNoAuth
	}
	return MethodNoAcceptable
}

// WriteMethod encodes the SOCKS5 method-selection reply.
func WriteMethod(writer io.Writer, method byte) error {
	if method != MethodNoAuth && method != MethodNoAcceptable {
		return fmt.Errorf("%w: %#x", ErrUnsupportedMethod, method)
	}
	return writeAll(writer, []byte{Version, method})
}

// ReadRequest decodes a SOCKS5 CONNECT request. Only CONNECT is accepted;
// BIND and UDP ASSOCIATE remain explicit future capabilities.
func ReadRequest(reader io.Reader) (Request, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return Request{}, fmt.Errorf("read SOCKS request header: %w", err)
	}
	if header[0] != Version {
		return Request{}, fmt.Errorf("%w: %#x", ErrUnsupportedVersion, header[0])
	}
	if header[1] != CommandConnect {
		return Request{}, fmt.Errorf("%w: %#x", ErrUnsupportedCommand, header[1])
	}
	if header[2] != 0 {
		return Request{}, fmt.Errorf("%w: %#x", ErrInvalidReservedByte, header[2])
	}

	host, err := readAddress(reader, header[3])
	if err != nil {
		return Request{}, err
	}
	var portWire [2]byte
	if _, err := io.ReadFull(reader, portWire[:]); err != nil {
		return Request{}, fmt.Errorf("read SOCKS destination port: %w", err)
	}
	port := binary.BigEndian.Uint16(portWire[:])
	if port == 0 {
		return Request{}, ErrInvalidPort
	}
	return Request{Command: CommandConnect, Host: host, Port: port}, nil
}

func readAddress(reader io.Reader, addressType byte) (string, error) {
	switch addressType {
	case AddressIPv4:
		var wire [4]byte
		if _, err := io.ReadFull(reader, wire[:]); err != nil {
			return "", fmt.Errorf("read SOCKS IPv4 address: %w", err)
		}
		return netip.AddrFrom4(wire).String(), nil
	case AddressIPv6:
		var wire [16]byte
		if _, err := io.ReadFull(reader, wire[:]); err != nil {
			return "", fmt.Errorf("read SOCKS IPv6 address: %w", err)
		}
		return netip.AddrFrom16(wire).String(), nil
	case AddressDomain:
		var length [1]byte
		if _, err := io.ReadFull(reader, length[:]); err != nil {
			return "", fmt.Errorf("read SOCKS domain length: %w", err)
		}
		if length[0] == 0 {
			return "", ErrEmptyDomain
		}
		wire := make([]byte, int(length[0]))
		if _, err := io.ReadFull(reader, wire); err != nil {
			return "", fmt.Errorf("read SOCKS domain: %w", err)
		}
		if !validDomain(wire) {
			return "", ErrInvalidDomain
		}
		return string(wire), nil
	default:
		return "", fmt.Errorf("%w: %#x", ErrUnsupportedAddressType, addressType)
	}
}

func validDomain(domain []byte) bool {
	if len(domain) == 0 || domain[0] == '.' || domain[len(domain)-1] == '.' {
		return false
	}

	labelStart := 0
	for i, character := range domain {
		if character == '.' {
			if i == labelStart || domain[i-1] == '-' {
				return false
			}
			labelStart = i + 1
			continue
		}
		if character == '-' {
			if i == labelStart {
				return false
			}
			continue
		}
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') {
			return false
		}
	}
	return true
}

// WriteReply encodes a SOCKS5 response with the unspecified bind address.
// The future relay owns any real bound-address reporting.
func WriteReply(writer io.Writer, reply byte) error {
	if !validReply(reply) {
		return fmt.Errorf("%w: %#x", ErrUnsupportedReply, reply)
	}
	return writeAll(writer, []byte{Version, reply, 0, AddressIPv4, 0, 0, 0, 0, 0, 0})
}

func validReply(reply byte) bool {
	return reply <= ReplyAddressNotSupported
}

func writeAll(writer io.Writer, bytes []byte) error {
	for len(bytes) > 0 {
		written, err := writer.Write(bytes)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		bytes = bytes[written:]
	}
	return nil
}
