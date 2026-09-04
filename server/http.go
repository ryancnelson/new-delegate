// Package server implements frontend protocol listeners and handlers.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/operation"
	"gitea.local/ryan/new-delegate/policy"
)

type fetchConnector interface {
	Fetch(context.Context, operation.Fetch) (operation.Result, error)
}

type httpHandler struct {
	server    string
	mounts    []mount.Mount
	rules     []policy.Rule
	connector fetchConnector
}

// NewHTTPHandler constructs an HTTP frontend for an already-validated runtime
// configuration.
func NewHTTPHandler(mounts []mount.Mount, rules []policy.Rule, connector fetchConnector) http.Handler {
	return NewHTTPHandlerForServer("", mounts, rules, connector)
}

// NewHTTPHandlerForServer constructs an HTTP frontend with a named-server
// identity used for scoped mount selection.
func NewHTTPHandlerForServer(server string, mounts []mount.Mount, rules []policy.Rule, connector fetchConnector) http.Handler {
	return &httpHandler{
		server:    server,
		mounts:    append([]mount.Mount(nil), mounts...),
		rules:     append([]policy.Rule(nil), rules...),
		connector: connector,
	}
}

func (h *httpHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	matched, err := mount.ResolveFor(h.mounts, mount.Request{
		Path: request.URL.EscapedPath(), Server: h.server, Protocol: "http",
	})
	if err != nil {
		switch {
		case errors.Is(err, mount.ErrNoMatch):
			http.Error(response, mount.ErrNoMatch.Error(), http.StatusNotFound)
		case errors.Is(err, mount.ErrUnsafePath):
			http.Error(response, mount.ErrUnsafePath.Error(), http.StatusBadRequest)
		default:
			http.Error(response, "mount resolution failed", http.StatusInternalServerError)
		}
		return
	}

	decision := policy.Evaluate(h.rules, policy.Request{
		Source:      remoteHost(request.RemoteAddr),
		Protocol:    "http",
		Destination: targetHostname(matched.Target),
		Method:      request.Method,
		Mount:       matched.Mount.Path,
	})

	var result operation.Result
	err = decision.Enforce(func() error {
		resource, buildErr := backendResource(matched.Target, request.URL.RawQuery)
		if buildErr != nil {
			return buildErr
		}
		result, buildErr = h.connector.Fetch(request.Context(), operation.Fetch{
			Method:   request.Method,
			Resource: resource,
			Metadata: cloneHeader(request.Header),
			Body:     request.Body,
		})
		return buildErr
	})
	if errors.Is(err, policy.ErrDenied) {
		http.Error(response, string(decision.Reason), http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(response, "backend fetch failed", http.StatusBadGateway)
		return
	}
	if result.Body != nil {
		defer result.Body.Close()
	}
	copyResponseHeader(response.Header(), result.Metadata)
	response.WriteHeader(result.Status)
	if result.Body != nil {
		_, _ = io.Copy(response, result.Body)
	}
}

func targetHostname(target string) string {
	parsed, err := url.Parse(target)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func backendResource(target, rawQuery string) (string, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("parse backend target: %w", err)
	}
	if rawQuery != "" {
		if parsed.RawQuery != "" {
			parsed.RawQuery += "&" + rawQuery
		} else {
			parsed.RawQuery = rawQuery
		}
	}
	return parsed.String(), nil
}

func remoteHost(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		return host
	}
	return address
}

func cloneHeader(source http.Header) map[string][]string {
	result := make(map[string][]string, len(source))
	for key, values := range source {
		result[key] = append([]string(nil), values...)
	}
	return result
}

var hopByHopHeaders = map[string]struct{}{
	"Connection":          {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func copyResponseHeader(destination http.Header, source map[string][]string) {
	connectionHeaders := make(map[string]struct{})
	for _, value := range source["Connection"] {
		for _, name := range strings.Split(value, ",") {
			connectionHeaders[http.CanonicalHeaderKey(strings.TrimSpace(name))] = struct{}{}
		}
	}
	for key, values := range source {
		canonical := http.CanonicalHeaderKey(key)
		if _, skip := hopByHopHeaders[canonical]; skip {
			continue
		}
		if _, skip := connectionHeaders[canonical]; skip {
			continue
		}
		for _, value := range values {
			destination.Add(canonical, value)
		}
	}
}
