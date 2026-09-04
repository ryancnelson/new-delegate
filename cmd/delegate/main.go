package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gitea.local/ryan/new-delegate/config"
	"gitea.local/ryan/new-delegate/connector"
	gatewayserver "gitea.local/ryan/new-delegate/server"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, serveHTTP))
}

func run(args []string, stdout, stderr io.Writer, serve func(config.Config) error) int {
	checkOnly := len(args) > 0 && args[0] == "check"
	if checkOnly || (len(args) > 0 && args[0] == "serve") {
		args = args[1:]
	}

	configured, err := config.ParseLegacyArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid configuration: %v\n", err)
		return 2
	}
	if err := configured.Validate(); err != nil {
		fmt.Fprintf(stderr, "invalid configuration: %v\n", err)
		return 2
	}
	if checkOnly {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(configured); err != nil {
			fmt.Fprintf(stderr, "write canonical configuration: %v\n", err)
			return 1
		}
		return 0
	}
	if len(configured.Servers) != 1 || !strings.EqualFold(configured.Servers[0].Protocol, "http") {
		fmt.Fprintln(stderr, "unsupported frontend protocol: the runnable slice currently supports exactly one HTTP server")
		return 2
	}
	if err := serve(configured); err != nil {
		fmt.Fprintf(stderr, "serve: %v\n", err)
		return 1
	}
	return 0
}

func serveHTTP(configured config.Config) error {
	backend := connector.NewHTTP(&http.Client{Timeout: 60 * time.Second})
	handler := gatewayserver.NewHTTPHandler(configured.Mounts, configured.Policies, backend)
	server := &http.Server{
		Addr:              configured.Servers[0].Listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
