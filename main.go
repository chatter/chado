package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chatter/chado/internal/app"
	"github.com/chatter/chado/internal/logger"
)

// version is set from build info or falls back to "dev"
var version string

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		v := info.Main.Version
		// Pseudo-versions are very long (40+ chars); real versions are short
		if len(v) > 20 {
			version = "(devel)"
		} else {
			version = v
		}
	}
}

func run(ctx context.Context, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("chado", flag.ContinueOnError)
	logLevel := fs.String("log-level", "", "log level: debug, info, warn, error")
	fs.StringVar(logLevel, "l", "", "log level (shorthand)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Initialize logger
	if err := logger.Init(*logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	defer logger.Close()

	if _, err := os.Stat(".jj"); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "error: not a jj repository (or any parent up to mount point /)")
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not get current directory: %v\n", err)
		return err
	}

	model := app.New(cwd, version)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return err
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
