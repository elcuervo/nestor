package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cretz/bine/tor"
)

var (
	styleLogo    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleBox     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 2).Foreground(lipgloss.Color("6"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
)

const logoText = `
               _
░▒▓███████▓▒░░▒▓████████▓▒░░▒▓███████▓▒░▒▓████████▓▒░▒▓██████▓▒░░▒▓███████▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░         ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░         ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓██████▓▒░  ░▒▓██████▓▒░   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓███████▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░             ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░             ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓███████▓▒░   ░▒▓█▓▒░   ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░

`

const indent = "  "
const tagLine = "NEtwork Share via TOR"
const madeBy = "made with ☠️ by elcuervo"

func indentBlock(s string) string {
	return indent + strings.ReplaceAll(s, "\n", "\n"+indent)
}

type phase int

const (
	phaseExtracting phase = iota
	phaseStartingTor
	phaseCreatingOnion
	phaseRunning
	phaseError
)

var phaseLabel = map[phase]string{
	phaseExtracting:    "Extracting Tor binary",
	phaseStartingTor:   "Starting Tor process",
	phaseCreatingOnion: "Creating onion service",
}

func phaseIsDone(done []phase, p phase) bool {
	for _, d := range done {
		if d == p {
			return true
		}
	}
	return false
}

type extractDoneMsg struct {
	torPath string
	dataDir string
	cleanup func()
	err     error
}

type torStartedMsg struct {
	t   *tor.Tor
	err error
}

type onionReadyMsg struct {
	onion *tor.OnionService
	err   error
}

type serveErrMsg struct{ err error }

type quitMsg struct{}

type model struct {
	port         int
	currentPhase phase
	donePhases   []phase
	spinner      spinner.Model
	torPath      string
	cleanup      func()
	t            *tor.Tor
	listenCancel context.CancelFunc
	onion        *tor.OnionService
	onionURL     string
	err          error
	width        int
}

func cmdExtractTor() tea.Cmd {
	return func() tea.Msg {
		torPath, dataDir, cleanup, err := extractTor()
		return extractDoneMsg{torPath: torPath, dataDir: dataDir, cleanup: cleanup, err: err}
	}
}

func cmdStartTor(path, dataDir string) tea.Cmd {
	return func() tea.Msg {
		t, err := startTor(path, dataDir)
		return torStartedMsg{t: t, err: err}
	}
}

func cmdCreateOnion(t *tor.Tor, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		onion, err := t.Listen(ctx, &tor.ListenConf{RemotePorts: []int{torRemotePort}, Version3: true})
		return onionReadyMsg{onion: onion, err: err}
	}
}

func cmdServe(onion *tor.OnionService, port int) tea.Cmd {
	return func() tea.Msg {
		var err error
		if port != 0 {
			err = proxyPort(onion, port)
		} else {
			err = http.Serve(onion, http.FileServer(http.Dir(".")))
		}
		return serveErrMsg{err: err}
	}
}

func cmdCleanupAndQuit(m model) tea.Cmd {
	return func() tea.Msg {
		if m.listenCancel != nil {
			m.listenCancel()
		}
		if m.onion != nil {
			m.onion.Close()
		}
		if m.t != nil {
			m.t.Close()
		}
		if m.cleanup != nil {
			m.cleanup()
		}
		return quitMsg{}
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, cmdExtractTor())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, cmdCleanupAndQuit(m)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case extractDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.currentPhase = phaseError
			return m, tea.Quit
		}
		m.torPath = msg.torPath
		m.cleanup = msg.cleanup
		m.donePhases = append(m.donePhases, phaseExtracting)
		m.currentPhase = phaseStartingTor
		return m, cmdStartTor(m.torPath, msg.dataDir)

	case torStartedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.currentPhase = phaseError
			return m, cmdCleanupAndQuit(m)
		}
		m.t = msg.t
		ctx, cancel := context.WithTimeout(context.Background(), onionCreateTimeout)
		m.listenCancel = cancel
		m.donePhases = append(m.donePhases, phaseStartingTor)
		m.currentPhase = phaseCreatingOnion
		return m, cmdCreateOnion(m.t, ctx)

	case onionReadyMsg:
		if msg.err != nil {
			m.err = msg.err
			m.currentPhase = phaseError
			return m, cmdCleanupAndQuit(m)
		}
		m.onion = msg.onion
		m.onionURL = fmt.Sprintf("http://%v.onion", msg.onion.ID)
		m.donePhases = append(m.donePhases, phaseCreatingOnion)
		m.currentPhase = phaseRunning
		return m, cmdServe(m.onion, m.port)

	case serveErrMsg:
		if msg.err != nil && !isClosedNetErr(msg.err) {
			m.err = msg.err
			m.currentPhase = phaseError
		}
		return m, tea.Quit

	case quitMsg:
		return m, tea.Quit
	}

	return m, nil
}

func (m model) View() string {
	lines := []string{
		indentBlock(styleLogo.Render(strings.TrimSpace(logoText))),
		"",
		indent + styleSuccess.Render(tagLine),
		"",
		indent + styleInfo.Render(madeBy),
		"",
	}

	for _, p := range []phase{phaseExtracting, phaseStartingTor, phaseCreatingOnion} {
		label := phaseLabel[p]
		if phaseIsDone(m.donePhases, p) {
			lines = append(lines, indent+styleSuccess.Render("✓ "+label))
		} else if m.currentPhase == p {
			lines = append(lines, indent+m.spinner.View()+" "+styleSuccess.Render(label))
		}
	}

	switch m.currentPhase {
	case phaseRunning:
		dir, err := os.Getwd()
		if err != nil {
			dir = "(unknown directory)"
		}
		location := styleInfo.Render("Serving " + dir)
		if m.port != 0 {
			location = styleInfo.Render(fmt.Sprintf("Forwarding localhost:%d → Tor", m.port))
		}
		lines = append(lines, "", indent+styleBox.Render(m.onionURL), indent+location, indent+styleInfo.Render("Press Ctrl+C to stop"), "")

	case phaseError:
		lines = append(lines, "", indent+styleError.Render(fmt.Sprintf("Error: %v", m.err)), "")
	}

	return strings.Join(lines, "\n")
}
