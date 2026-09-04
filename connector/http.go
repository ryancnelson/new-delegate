// Package connector implements backend protocol clients.
package connector

import (
	"context"
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
	return &HTTP{client: client}
}

func (h *HTTP) Fetch(ctx context.Context, fetch operation.Fetch) (operation.Result, error) {
	request, err := http.NewRequestWithContext(ctx, fetch.Method, fetch.Resource, fetch.Body)
	if err != nil {
		return operation.Result{}, err
	}
	request.Header = cloneMetadata(fetch.Metadata)

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
