// Package connector implements backend protocol clients.
package connector

import (
	"context"
	"io"
	"net/http"

	"gitea.local/ryan/new-delegate/operation"
)

// HTTP executes Fetch operations using an HTTP backend.
type HTTP struct {
	client *http.Client
}

func NewHTTP(client *http.Client) *HTTP {
	if client == nil {
		client = http.DefaultClient
	}
	private := *client
	private.CheckRedirect = stopRedirects
	return &HTTP{client: &private}
}

func stopRedirects(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

func (h *HTTP) Fetch(ctx context.Context, fetch operation.Fetch) (operation.Result, error) {
	return h.execute(ctx, fetch.Method, fetch.Resource, fetch.Metadata, fetch.Body, -1)
}

// Store executes one semantic write operation using an HTTP backend.
func (h *HTTP) Store(ctx context.Context, store operation.Store) (operation.Result, error) {
	return h.execute(ctx, store.Method, store.Resource, store.Metadata, store.Body, store.Size)
}

func (h *HTTP) execute(ctx context.Context, method, resource string, metadata map[string][]string, body io.Reader, size int64) (operation.Result, error) {
	request, err := http.NewRequestWithContext(ctx, method, resource, body)
	if err != nil {
		return operation.Result{}, err
	}
	request.Header = cloneMetadata(metadata)
	if size >= 0 {
		request.ContentLength = size
	}

	response, err := h.client.Do(request)
	if err != nil {
		return operation.Result{}, err
	}
	return operation.Result{
		Status:   response.StatusCode,
		Metadata: cloneMetadata(response.Header),
		Body:     response.Body,
	}, nil
}

func cloneMetadata(source map[string][]string) map[string][]string {
	result := make(map[string][]string, len(source))
	for key, values := range source {
		result[key] = append([]string(nil), values...)
	}
	return result
}
