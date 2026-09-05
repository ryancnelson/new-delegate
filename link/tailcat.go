package link

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/tailscale/tailcat"
)

const pairingPrefix = "ndlink1"

// Pairing contains the one-run Tailcat address and the fixed TCP port exposed
// by the right side. It is a capability secret and must not be logged or
// persisted.
type Pairing struct {
	RemotePort uint16
	Address    string
}

// Encode returns the single-line handoff form consumed by ParsePairing.
func (p Pairing) Encode() string {
	return fmt.Sprintf("%s:%d:%s", pairingPrefix, p.RemotePort, p.Address)
}

// ParsePairing validates the one-line, in-memory handoff form.
func ParsePairing(value string) (Pairing, error) {
	if value != strings.TrimSpace(value) {
		return Pairing{}, errors.New("invalid link pairing whitespace")
	}
	parts := strings.Split(value, ":")
	if len(parts) != 3 || parts[0] != pairingPrefix {
		return Pairing{}, errors.New("invalid link pairing")
	}
	port, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil || port == 0 {
		return Pairing{}, errors.New("invalid link pairing port")
	}
	if !validTailcatAddress(parts[2]) {
		return Pairing{}, errors.New("invalid link pairing address")
	}
	return Pairing{RemotePort: uint16(port), Address: parts[2]}, nil
}

func validTailcatAddress(address string) bool {
	_, err := tailcat.ParseAddr(tailcat.Addr(address))
	return err == nil
}
