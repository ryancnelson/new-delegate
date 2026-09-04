package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gitea.local/ryan/new-delegate/release"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: release VERSION OUTPUT_DIRECTORY")
		os.Exit(2)
	}
	version, outputDirectory := os.Args[1], os.Args[2]
	sourceRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "release: determine source root: %v\n", err)
		os.Exit(1)
	}
	build := func(ctx context.Context, target release.Target, output, stampedVersion string) error {
		command := exec.CommandContext(ctx, "go", "build",
			"-trimpath", "-buildvcs=false",
			"-ldflags=-s -w -buildid= -X main.version="+stampedVersion,
			"-o", output, "./cmd/delegate",
		)
		command.Dir = sourceRoot
		command.Env = releaseEnvironment(os.Environ(), target)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		return command.Run()
	}
	if err := release.Build(context.Background(), sourceRoot, outputDirectory, version, build); err != nil {
		fmt.Fprintf(os.Stderr, "release: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, outputDirectory)
}

func releaseEnvironment(current []string, target release.Target) []string {
	result := make([]string, 0, len(current)+3)
	for _, entry := range current {
		if strings.HasPrefix(entry, "CGO_ENABLED=") || strings.HasPrefix(entry, "GOOS=") || strings.HasPrefix(entry, "GOARCH=") {
			continue
		}
		result = append(result, entry)
	}
	return append(result, "CGO_ENABLED=0", "GOOS="+target.GOOS, "GOARCH="+target.GOARCH)
}
