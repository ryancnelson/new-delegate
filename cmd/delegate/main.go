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
	"gitea.local/ryan/new-delegate/explain"
	gatewayserver "gitea.local/ryan/new-delegate/server"
)

func main() {
	args := os.Args[1:]
	reloadPath := configPathFromArgs(args)
	os.Exit(run(args, os.Stdout, os.Stderr, func(configured config.Config) error {
		return serveHTTP(configured, reloadPath, os.Stderr)
	}))
}

func configPathFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--config=") {
			return strings.TrimPrefix(arg, "--config=")
		}
	}
	return ""
}

func run(args []string, stdout, stderr io.Writer, serve func(config.Config) error) int {
	mode := "serve"
	if len(args) > 0 && (args[0] == "check" || args[0] == "serve" || args[0] == "explain") {
		mode = args[0]
		args = args[1:]
	}

	if mode == "explain" {
		configured, request, err := loadExplanation(args)
		if err != nil {
			fmt.Fprintf(stderr, "invalid explanation: %v\n", err)
			return 2
		}
		result, err := explain.Evaluate(configured, request)
		if err != nil {
			fmt.Fprintf(stderr, "explain: %v\n", err)
			return 2
		}
		if err := writeJSON(stdout, result); err != nil {
			fmt.Fprintf(stderr, "write explanation: %v\n", err)
			return 1
		}
		return 0
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
	if mode == "check" {
		if err := writeJSON(stdout, configured); err != nil {
			fmt.Fprintf(stderr, "write canonical configuration: %v\n", err)
			return 1
		}
		return 0
	}
	if err := validateRuntimeSupport(configured); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := serve(configured); err != nil {
		fmt.Fprintf(stderr, "serve: %v\n", err)
		return 1
	}
	return 0
}

func validateRuntimeSupport(configured config.Config) error {
	for _, frontend := range configured.Servers {
		if !strings.EqualFold(frontend.Protocol, "http") {
			return fmt.Errorf("unsupported frontend protocol: the runnable slice currently supports HTTP servers")
		}
		if frontend.TLS != nil {
			return fmt.Errorf("unsupported TLS runtime: frontend TLS is configured but not implemented")
		}
	}
	for _, mounted := range configured.Mounts {
		if mounted.TLS != nil {
			return fmt.Errorf("unsupported TLS runtime: backend TLS policy is configured but not implemented")
		}
	}
	return nil
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func loadExplanation(args []string) (config.Config, explain.Request, error) {
	var request explain.Request
	var configArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		value := func(name string) (string, error) {
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return "", fmt.Errorf("%s requires a value", name)
			}
			i++
			return args[i], nil
		}
		switch {
		case arg == "--path":
			parsed, err := value("--path")
			if err != nil {
				return config.Config{}, explain.Request{}, err
			}
			request.Path = parsed
		case strings.HasPrefix(arg, "--path="):
			request.Path = strings.TrimPrefix(arg, "--path=")
		case arg == "--source":
			parsed, err := value("--source")
			if err != nil {
				return config.Config{}, explain.Request{}, err
			}
			request.Source = parsed
		case strings.HasPrefix(arg, "--source="):
			request.Source = strings.TrimPrefix(arg, "--source=")
		case arg == "--method":
			parsed, err := value("--method")
			if err != nil {
				return config.Config{}, explain.Request{}, err
			}
			request.Method = parsed
		case strings.HasPrefix(arg, "--method="):
			request.Method = strings.TrimPrefix(arg, "--method=")
		case arg == "--config":
			parsed, err := value("--config")
			if err != nil {
				return config.Config{}, explain.Request{}, err
			}
			configArgs = append(configArgs, "--config", parsed)
		case strings.HasPrefix(arg, "--config="):
			configArgs = append(configArgs, arg)
		default:
			configArgs = append(configArgs, arg)
		}
	}
	if strings.TrimSpace(request.Path) == "" {
		return config.Config{}, explain.Request{}, fmt.Errorf("--path is required")
	}
	if strings.TrimSpace(request.Source) == "" {
		return config.Config{}, explain.Request{}, fmt.Errorf("--source is required")
	}
	if strings.TrimSpace(request.Method) == "" {
		return config.Config{}, explain.Request{}, fmt.Errorf("--method is required")
	}

	configured, err := loadConfiguration(configArgs)
	if err != nil {
		return config.Config{}, explain.Request{}, err
	}
	if len(configured.Servers) != 1 {
		return config.Config{}, explain.Request{}, fmt.Errorf("explain currently requires exactly one configured server")
	}
	request.Protocol = configured.Servers[0].Protocol
	request.Server = configured.Servers[0].Name
	return configured, request, nil
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

func serveHTTP(configured config.Config, reloadPath string, logOutput io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	backend := connector.NewHTTP(&http.Client{Timeout: 60 * time.Second})
	configStore, err := config.NewStore(configured)
	if err != nil {
		return fmt.Errorf("initialize configuration store: %w", err)
	}
	if reloadPath != "" {
		signals := reloadSignals()
		if len(signals) > 0 {
			reloadEvents := make(chan os.Signal, 1)
			signal.Notify(reloadEvents, signals...)
			defer signal.Stop(reloadEvents)
			go watchReload(ctx, reloadEvents, configStore, reloadPath, func(err error) {
				if err != nil {
					fmt.Fprintf(logOutput, "configuration reload rejected: %v\n", err)
					return
				}
				fmt.Fprintln(logOutput, "configuration reloaded")
			})
		}
	}
	httpServers := make([]*http.Server, 0, len(configured.Servers))
	listeners := make([]net.Listener, 0, len(configured.Servers))
	for _, frontend := range configured.Servers {
		handler := gatewayserver.NewReloadableHTTPHandler(frontend.Name, configStore.Snapshot, backend)
		httpServer := &http.Server{
			Addr:              frontend.Listen,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       2 * time.Minute,
		}
		listener, err := net.Listen("tcp", httpServer.Addr)
		if err != nil {
			for _, opened := range listeners {
				_ = opened.Close()
			}
			return fmt.Errorf("listen for server %q: %w", frontend.Name, err)
		}
		httpServers = append(httpServers, httpServer)
		listeners = append(listeners, listener)
	}
	return gatewayserver.ServeAll(ctx, httpServers, listeners, 10*time.Second)
}

func watchReload(ctx context.Context, events <-chan os.Signal, store *config.Store, path string, report func(error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-events:
			report(config.ReloadTOMLFileWithValidation(store, path, validateRuntimeSupport))
		}
	}
}
