package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	configured, err := loadConfiguration(args)
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

func loadConfiguration(args []string) (config.Config, error) {
	configIndex := -1
	for i, arg := range args {
		if arg == "--config" || strings.HasPrefix(arg, "--config=") {
			configIndex = i
			break
		}
	}
	if configIndex == -1 {
		return config.ParseLegacyArgs(args)
	}
	if configIndex != 0 {
		return config.Config{}, fmt.Errorf("cannot combine --config with legacy directives")
	}

	var path string
	if args[0] == "--config" {
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			return config.Config{}, fmt.Errorf("--config requires a path")
		}
		path = args[1]
		if len(args) != 2 {
			return config.Config{}, fmt.Errorf("cannot combine --config with legacy directives")
		}
	} else {
		path = strings.TrimSpace(strings.TrimPrefix(args[0], "--config="))
		if path == "" {
			return config.Config{}, fmt.Errorf("--config requires a path")
		}
		if len(args) != 1 {
			return config.Config{}, fmt.Errorf("cannot combine --config with legacy directives")
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("read configuration %q: %w", path, err)
	}
	defer file.Close()

	configured, err := config.ParseTOML(file)
	if err != nil {
		return config.Config{}, err
	}
	return configured, nil
}

func serveHTTP(configured config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	backend := connector.NewHTTP(&http.Client{Timeout: 60 * time.Second})
	handler := gatewayserver.NewHTTPHandler(configured.Mounts, configured.Policies, backend)
	httpServer := &http.Server{
		Addr:              configured.Servers[0].Listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	listener, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return err
	}
	return gatewayserver.Serve(ctx, httpServer, listener, 10*time.Second)
}
