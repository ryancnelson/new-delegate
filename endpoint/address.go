// Package endpoint parses the side-effect-free, socat-style two-address
// grammar used to describe transparent byte-stream routes.
package endpoint

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"
)

// Kind identifies an endpoint type. The spelling is deliberately the same as
// the command-line grammar so diagnostics and canonical output stay familiar.
type Kind string

const (
	TCPListen      Kind = "TCP-LISTEN"
	TCPConnect     Kind = "TCP-CONNECT"
	TailcatListen  Kind = "TAILCAT-LISTEN"
	TailcatConnect Kind = "TAILCAT-CONNECT"
)

// Input identifies a source from which sensitive endpoint material is read.
type Input string

const Stdin Input = "@stdin"

// Address is the parsed, side-effect-free form of one command-line address.
// Fields that do not apply to the selected Kind remain zero-valued.
type Address struct {
	Kind  Kind
	Host  string
	Port  uint16
	Input Input
}

// Route connects one accepting address to one dialing address.
type Route struct {
	Ingress Address
	Egress  Address
}

// ParseAddress parses one strict address. Address keywords are case-sensitive,
// options are reserved for later typed schemas, and parsing performs no I/O.
func ParseAddress(value string) (Address, error) {
	if value == "" {
		return Address{}, fmt.Errorf("address is required")
	}
	if value != strings.TrimSpace(value) {
		return Address{}, fmt.Errorf("address must not contain surrounding whitespace")
	}

	rawKind, argument, found := strings.Cut(value, ":")
	if !found {
		return Address{}, fmt.Errorf("unknown address type %q", value)
	}
	kind := Kind(rawKind)
	switch kind {
	case TCPListen, TCPConnect:
		if strings.Contains(argument, ",") {
			return Address{}, fmt.Errorf("%s options are not supported yet", kind)
		}
		host, rawPort, err := net.SplitHostPort(argument)
		if err != nil {
			return Address{}, fmt.Errorf("%s must use host:port: %w", kind, err)
		}
		if host == "" {
			return Address{}, fmt.Errorf("%s host is required", kind)
		}
		if strings.IndexFunc(host, unicode.IsSpace) >= 0 {
			return Address{}, fmt.Errorf("%s has invalid host %q", kind, host)
		}
		port, err := parsePort(kind, rawPort)
		if err != nil {
			return Address{}, err
		}
		return Address{Kind: kind, Host: host, Port: port}, nil

	case TailcatListen:
		if strings.Contains(argument, ",") {
			return Address{}, fmt.Errorf("%s options are not supported yet", kind)
		}
		port, err := parsePort(kind, argument)
		if err != nil {
			return Address{}, err
		}
		return Address{Kind: kind, Port: port}, nil

	case TailcatConnect:
		if strings.Contains(argument, ",") {
			return Address{}, fmt.Errorf("%s options are not supported yet", kind)
		}
		if argument != string(Stdin) {
			return Address{}, fmt.Errorf("%s handoff source must be @stdin", kind)
		}
		return Address{Kind: kind, Input: Stdin}, nil

	default:
		return Address{}, fmt.Errorf("unknown address type %q", rawKind)
	}
}

// ParseRoute parses exactly two addresses and checks their directional roles.
func ParseRoute(args []string) (Route, error) {
	if len(args) != 2 {
		return Route{}, fmt.Errorf("route requires exactly two addresses")
	}
	ingress, err := ParseAddress(args[0])
	if err != nil {
		return Route{}, fmt.Errorf("first address: %w", err)
	}
	if !ingress.listens() {
		return Route{}, fmt.Errorf("first address must listen, got %s", ingress.Kind)
	}
	egress, err := ParseAddress(args[1])
	if err != nil {
		return Route{}, fmt.Errorf("second address: %w", err)
	}
	if !egress.connects() {
		return Route{}, fmt.Errorf("second address must connect, got %s", egress.Kind)
	}
	return Route{Ingress: ingress, Egress: egress}, nil
}

func parsePort(kind Kind, value string) (uint16, error) {
	parsed, err := strconv.ParseUint(value, 10, 16)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("%s has invalid port %q", kind, value)
	}
	return uint16(parsed), nil
}

func (a Address) listens() bool {
	return a.Kind == TCPListen || a.Kind == TailcatListen
}

func (a Address) connects() bool {
	return a.Kind == TCPConnect || a.Kind == TailcatConnect
}
