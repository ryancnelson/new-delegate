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

// Result is a connector's response to an operation.
type Result struct {
	Status   int
	Metadata map[string][]string
	Body     io.ReadCloser
}
