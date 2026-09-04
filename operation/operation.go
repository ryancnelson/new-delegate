// Package operation defines protocol-neutral requests and results exchanged by
// frontends and backend connectors.
package operation

import "io"

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
	Status   int
	Metadata map[string][]string
	Body     io.ReadCloser
}
