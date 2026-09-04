package link

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
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
	value = strings.TrimSpace(value)
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

// TailcatServeArgs returns the fixed, loopback-only Tailcat serve invocation.
func TailcatServeArgs(target string) ([]string, uint16, error) {
	port, err := loopbackPort(target)
	if err != nil {
		return nil, 0, err
	}
	return []string{"serve", strconv.Itoa(int(port))}, port, nil
}

// TailcatForwardArgs returns the fixed, loopback-only Tailcat forward
// invocation. The pairing address is intentionally supplied only to the
// short-lived test-mode Tailcat child process.
func TailcatForwardArgs(pairing Pairing, listen string) ([]string, error) {
	localPort, err := loopbackPort(listen)
	if err != nil {
		return nil, err
	}
	if pairing.RemotePort == 0 || !validTailcatAddress(pairing.Address) {
		return nil, errors.New("invalid link pairing")
	}
	mapping := fmt.Sprintf("%d:%d", localPort, pairing.RemotePort)
	return []string{"forward", pairing.Address, mapping}, nil
}

// RunRight starts Tailcat's right-side service and writes exactly one pairing
// handoff line when Tailcat announces its ephemeral address. Tailcat output
// containing the address is deliberately not forwarded to diagnostics.
func RunRight(ctx context.Context, binary, target string, pairingOutput, diagnostics io.Writer) error {
	args, port, err := TailcatServeArgs(target)
	if err != nil {
		return err
	}
	if strings.TrimSpace(binary) == "" {
		return errors.New("tailcat binary is required")
	}
	if pairingOutput == nil || diagnostics == nil {
		return errors.New("link output is required")
	}

	command := exec.CommandContext(ctx, binary, args...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		return err
	}

	lines := make(chan string)
	var readers sync.WaitGroup
	readLines := func(reader io.Reader) {
		defer readers.Done()
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}
	readers.Add(2)
	go readLines(stdout)
	go readLines(stderr)
	go func() {
		readers.Wait()
		close(lines)
	}()

	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	pairingWritten := false
	for lines != nil || done != nil {
		select {
		case line, ok := <-lines:
			if !ok {
				lines = nil
				continue
			}
			if address := tailcatAddressIn(line); address != "" {
				if !pairingWritten {
					if _, err := fmt.Fprintln(pairingOutput, Pairing{RemotePort: port, Address: address}.Encode()); err != nil {
						return err
					}
					pairingWritten = true
				}
				continue
			}
			if strings.TrimSpace(line) != "" {
				_, _ = fmt.Fprintln(diagnostics, line)
			}
		case err := <-done:
			done = nil
			if ctx.Err() != nil {
				return nil
			}
			if err != nil {
				return fmt.Errorf("tailcat serve: %w", err)
			}
			if !pairingWritten {
				return errors.New("tailcat serve exited without a pairing address")
			}
			return nil
		case <-ctx.Done():
			// CommandContext terminates the child; the done case returns once it
			// has been reaped.
		}
	}
	if !pairingWritten {
		return errors.New("tailcat serve exited without a pairing address")
	}
	return nil
}

// RunLeft consumes a one-line pairing handoff and runs Tailcat's local
// loopback forwarder until ctx is canceled.
func RunLeft(ctx context.Context, binary string, pairingInput io.Reader, listen string, diagnostics io.Writer) error {
	if strings.TrimSpace(binary) == "" {
		return errors.New("tailcat binary is required")
	}
	if pairingInput == nil || diagnostics == nil {
		return errors.New("link input is required")
	}
	encoded, err := io.ReadAll(io.LimitReader(pairingInput, 4096))
	if err != nil {
		return fmt.Errorf("read link pairing: %w", err)
	}
	pairing, err := ParsePairing(string(encoded))
	if err != nil {
		return err
	}
	args, err := TailcatForwardArgs(pairing, listen)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, binary, args...)
	command.Stdout = diagnostics
	command.Stderr = diagnostics
	err = command.Run()
	if ctx.Err() != nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("tailcat forward: %w", err)
	}
	return nil
}

func loopbackPort(address string) (uint16, error) {
	host, rawPort, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return 0, fmt.Errorf("invalid loopback address %q", address)
	}
	if host != "localhost" {
		parsed := net.ParseIP(host)
		if parsed == nil || !parsed.IsLoopback() {
			return 0, fmt.Errorf("address %q is not loopback", address)
		}
	}
	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil || port == 0 {
		return 0, fmt.Errorf("invalid port in %q", address)
	}
	return uint16(port), nil
}

func validTailcatAddress(address string) bool {
	if len(address) < 3 || !strings.HasPrefix(address, "tc") {
		return false
	}
	for _, character := range address {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func tailcatAddressIn(line string) string {
	for _, field := range strings.Fields(line) {
		candidate := strings.Trim(field, ".,:;()[]{}")
		if validTailcatAddress(candidate) {
			return candidate
		}
	}
	return ""
}
