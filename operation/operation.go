// Package operation defines protocol-neutral requests and results exchanged by
// frontends and backend connectors.
package operation

import (
	"errors"
	"io"
)

// ErrUnsupported reports that a connector cannot perform the requested
// semantic operation. Frontends may translate it without contacting a backend.
var ErrUnsupported = errors.New("operation unsupported")

// Outcome describes a protocol-neutral operation result. HTTP connectors use
// OutcomePassthrough with Status; translated protocols select a semantic value.
type Outcome uint8

const (
	OutcomePassthrough Outcome = iota
	OutcomeSuccess
	OutcomeNotFound
	OutcomePermissionDenied
	OutcomeUpstreamFailure
)

// Fetch asks a connector to retrieve one resource.
type Fetch struct {
	Method   string
	Resource string
	Metadata map[string][]string
	Body     io.Reader
}

// Store asks a connector to write one resource.
type Store struct {
	Method   string
	Resource string
	Metadata map[string][]string
	Body     io.Reader
	Size     int64
}

// Relay asks a connector to open one bounded transparent byte stream. It is
// intentionally separate from semantic Fetch translation.
type Relay struct {
	Network string
	Address string
}

// Result is a connector's response to an operation.
type Result struct {
	Outcome  Outcome
	Status   int
	Metadata map[string][]string
	Body     io.ReadCloser
}
