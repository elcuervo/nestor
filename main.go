package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func validatePort(port int) error {
	if port == 0 {
		return nil
	}
	if port < minPort || port > maxPort {
		return fmt.Errorf("port %d is out of range: must be between %d and %d", port, minPort, maxPort)
	}
	return nil
}

func main() {
	var port int
	var quiet bool
	flag.IntVar(&port, "port", 0, "Local port to forward through Tor (default: serve current directory)")
	flag.IntVar(&port, "p", 0, "Local port to forward (shorthand)")
	flag.BoolVar(&quiet, "quiet", false, "Only print the onion URL")
	flag.BoolVar(&quiet, "q", false, "Only print the onion URL (shorthand)")
	flag.Parse()

	if err := validatePort(port); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if quiet {
		runQuiet(port)
		return
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	m := model{
		port:         port,
		currentPhase: phaseExtracting,
		spinner:      s,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fm, ok := finalModel.(model); ok && fm.err != nil {
		os.Exit(1)
	}
}
