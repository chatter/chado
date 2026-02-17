// Package main is the entry point for chado, a TUI for jj repositories.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	tea "charm.land/bubbletea/v2"
	"github.com/chatter/chado/internal/app"
	"github.com/chatter/chado/internal/logger"
)

// maxRealVersionLen is the upper bound for a "real" semver tag.
// Pseudo-versions are very long (40+ chars); real versions are short.
const maxRealVersionLen = 20

// resolveVersion returns the module version from build info, or "".
func resolveVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return ""
	}

	if len(info.Main.Version) > maxRealVersionLen {
		return "(devel)"
	}

	return info.Main.Version
}

func run(ctx context.Context, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("chado", flag.ContinueOnError)
	logLevel := fs.String("log-level", "", "log level: debug, info, warn, error")
	fs.StringVar(logLevel, "l", "", "log level (shorthand)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	// Initialize logger
	log, err := logger.New(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		// Create no-op logger so we can continue
		log, _ = logger.New("")
	}
	defer log.Close()

	if _, err := os.Stat(".jj"); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "error: not a jj repository (or any parent up to mount point /)")
		return fmt.Errorf("checking jj repository: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not get current directory: %v\n", err)
		return fmt.Errorf("getting working directory: %w", err)
	}

	version := resolveVersion()
	model := app.New(cwd, version, log)

	p := tea.NewProgram(
		model,
		tea.WithContext(ctx),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return fmt.Errorf("running program: %w", err)
	}

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
