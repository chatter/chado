package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/chatter/lazyjj/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

// version is set from build info or falls back to "dev"
var version = "dev"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}
}

func main() {
	// Check if we're in a jj repo
	if _, err := os.Stat(".jj"); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "error: not a jj repository (or any parent up to mount point /)")
		os.Exit(1)
	}

	// Get current working directory for the watcher
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not get current directory: %v\n", err)
		os.Exit(1)
	}

	// Create the app model
	model := app.New(cwd, version)

	// Create and run the BubbleTea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
