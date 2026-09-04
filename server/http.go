// Package server implements frontend protocol listeners and handlers.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"gitea.local/ryan/new-delegate/clientaddr"
	"gitea.local/ryan/new-delegate/config"
	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/operation"
	"gitea.local/ryan/new-delegate/policy"
)

type fetchConnector interface {
	Fetch(context.Context, operation.Fetch) (operation.Result, error)
}

type mountFetchConnector interface {
	FetchForMount(context.Context, mount.Mount, operation.Fetch) (operation.Result, error)
}

type httpHandler struct {
	server    string
	snapshot  func() config.Config
	connector mountFetchConnector
}

// NewHTTPHandler constructs an HTTP frontend for an already-validated runtime
// configuration.
func NewHTTPHandler(mounts []mount.Mount, rules []policy.Rule, connector fetchConnector) http.Handler {
	return NewHTTPHandlerForServer("", mounts, rules, connector)
}

// NewHTTPHandlerForServer constructs an HTTP frontend with a named-server
// identity used for scoped mount selection.
func NewHTTPHandlerForServer(server string, mounts []mount.Mount, rules []policy.Rule, connector fetchConnector) http.Handler {
	fixed := config.Config{
		Mounts: append([]mount.Mount(nil), mounts...), Policies: append([]policy.Rule(nil), rules...),
	}
	return NewReloadableHTTPHandler(server, func() config.Config { return fixed }, connector)
}

// NewReloadableHTTPHandler reads exactly one complete configuration snapshot
// for each request, keeping routing and policy evaluation on the same version.
func NewReloadableHTTPHandler(server string, snapshot func() config.Config, connector fetchConnector) http.Handler {
	return NewReloadableHTTPHandlerWithRoutes(server, snapshot, fixedMountConnector{connector})
}

// NewReloadableHTTPHandlerWithRoutes passes the resolver-selected mount to a
// connector router without adding transport policy to the semantic operation.
func NewReloadableHTTPHandlerWithRoutes(server string, snapshot func() config.Config, connector mountFetchConnector) http.Handler {
	return &httpHandler{
		server:    server,
		snapshot:  snapshot,
		connector: connector,
	}
}

func (h *httpHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.URL.IsAbs() && (request.URL.User != nil || request.URL.Fragment != "") {
		http.Error(response, "invalid absolute request target", http.StatusBadRequest)
		return
	}
	configured := h.snapshot()
	matched, err := mount.ResolveFor(configured.Mounts, mount.Request{
		Path: request.URL.EscapedPath(), Scheme: request.URL.Scheme,
		Authority: request.URL.Host, Server: h.server, Protocol: "http",
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

	source, err := policySource(configured, h.server, request)
	if err != nil {
		http.Error(response, "invalid client address", http.StatusBadRequest)
		return
	}
	decision := policy.Evaluate(configured.Policies, policy.Request{
		Source:      source,
		Protocol:    "http",
		Destination: targetHostname(matched.Target),
		Method:      request.Method,
		Mount:       matched.Mount.Pattern(),
	})

	var result operation.Result
	err = decision.Enforce(func() error {
		resource, buildErr := backendResource(matched.Target, request.URL.RawQuery)
		if buildErr != nil {
			return buildErr
		}
		result, buildErr = h.connector.FetchForMount(request.Context(), matched.Mount, operation.Fetch{
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

type fixedMountConnector struct {
	fetchConnector
}

func (f fixedMountConnector) FetchForMount(ctx context.Context, _ mount.Mount, fetch operation.Fetch) (operation.Result, error) {
	return f.Fetch(ctx, fetch)
}

func policySource(configured config.Config, serverName string, request *http.Request) (string, error) {
	for _, frontend := range configured.Servers {
		if frontend.Name != serverName || frontend.ClientIPHeader == "" {
			continue
		}
		prefixes := make([]netip.Prefix, 0, len(frontend.TrustedProxies))
		for _, text := range frontend.TrustedProxies {
			prefix, err := netip.ParsePrefix(text)
			if err != nil {
				return "", err
			}
			prefixes = append(prefixes, prefix)
		}
		forwarded := strings.Join(request.Header.Values(frontend.ClientIPHeader), ",")
		return clientaddr.Resolve(request.RemoteAddr, forwarded, prefixes)
	}
	return remoteHost(request.RemoteAddr), nil
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
	return sanitizeHeader(source)
}

var hopByHopHeaders = map[string]struct{}{
	"Connection":          {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Proxy-Connection":    {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func sanitizeHeader(source map[string][]string) map[string][]string {
	connectionHeaders := make(map[string]struct{})
	for key, values := range source {
		if !strings.EqualFold(key, "Connection") {
			continue
		}
		for _, value := range values {
			for _, name := range strings.Split(value, ",") {
				canonical := http.CanonicalHeaderKey(strings.TrimSpace(name))
				if canonical != "" {
					connectionHeaders[canonical] = struct{}{}
				}
			}
		}
	}
	result := make(map[string][]string, len(source))
	for key, values := range source {
		canonical := http.CanonicalHeaderKey(key)
		if canonical == "" {
			continue
		}
		if _, skip := hopByHopHeaders[canonical]; skip {
			continue
		}
		if _, skip := connectionHeaders[canonical]; skip {
			continue
		}
		result[canonical] = append(result[canonical], values...)
	}
	return result
}

func copyResponseHeader(destination http.Header, source map[string][]string) {
	for key, values := range sanitizeHeader(source) {
		destination[key] = append(destination[key], values...)
	}
}
