package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cretz/bine/tor"
	"github.com/ulikunitz/xz"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	styleLogo    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleBox     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 2).Foreground(lipgloss.Color("6"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
)

// ── Logo ──────────────────────────────────────────────────────────────────────

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

const tagLine = "NEtwork Share via TOR"

// ── Phase ─────────────────────────────────────────────────────────────────────

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

// ── Messages ──────────────────────────────────────────────────────────────────

type extractDoneMsg struct {
	torPath string
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

// ── Model ─────────────────────────────────────────────────────────────────────

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

// ── Commands ──────────────────────────────────────────────────────────────────

func cmdExtractTor() tea.Cmd {
	return func() tea.Msg {
		torPath, cleanup, err := extractTor()
		return extractDoneMsg{torPath: torPath, cleanup: cleanup, err: err}
	}
}

func cmdStartTor(path string) tea.Cmd {
	return func() tea.Msg {
		t, err := tor.Start(nil, &tor.StartConf{ExePath: path})
		return torStartedMsg{t: t, err: err}
	}
}

func cmdCreateOnion(t *tor.Tor, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		onion, err := t.Listen(ctx, &tor.ListenConf{RemotePorts: []int{80}, Version3: true})
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

// ── Init ──────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, cmdExtractTor())
}

// ── Update ────────────────────────────────────────────────────────────────────

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
		return m, cmdStartTor(m.torPath)

	case torStartedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.currentPhase = phaseError
			return m, cmdCleanupAndQuit(m)
		}
		m.t = msg.t
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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

// ── View ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	var b strings.Builder

	b.WriteString(styleLogo.Render(strings.TrimSpace(logoText)))
	b.WriteString("\n")
	b.WriteString(styleSuccess.Render(tagLine))
	b.WriteString("\n\n")

	phases := []phase{phaseExtracting, phaseStartingTor, phaseCreatingOnion}
	for _, p := range phases {
		label := phaseLabel[p]
		if phaseIsDone(m.donePhases, p) {
			b.WriteString(styleSuccess.Render("✓ " + label))
			b.WriteString("\n")
		} else if m.currentPhase == p {
			b.WriteString(m.spinner.View() + " " + styleSuccess.Render(label) + "\n")
		}
	}

	switch m.currentPhase {
	case phaseRunning:
		b.WriteString("\n")
		b.WriteString(styleBox.Render(m.onionURL))
		b.WriteString("\n")
		if m.port != 0 {
			b.WriteString(styleInfo.Render(fmt.Sprintf("Forwarding localhost:%d → Tor", m.port)))
		} else {
			dir, _ := os.Getwd()
			b.WriteString(styleInfo.Render("Serving " + dir))
		}
		b.WriteString("\n")
		b.WriteString(styleInfo.Render("Press Ctrl+C to stop"))
		b.WriteString("\n")

	case phaseError:
		b.WriteString("\n")
		b.WriteString(styleError.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	return b.String()
}

// ── Quiet mode ────────────────────────────────────────────────────────────────

func runQuiet(port int) {
	torPath, cleanup, err := extractTor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	t, err := tor.Start(nil, &tor.StartConf{ExePath: torPath})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer t.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	onion, err := t.Listen(ctx, &tor.ListenConf{RemotePorts: []int{80}, Version3: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer onion.Close()

	fmt.Printf("http://%v.onion\n", onion.ID)

	if port != 0 {
		proxyPort(onion, port)
	} else {
		http.Serve(onion, http.FileServer(http.Dir(".")))
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func isClosedNetErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "use of closed network connection")
}

func decompress(data []byte) ([]byte, error) {
	r, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func proxyPort(l net.Listener, port int) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go func(c net.Conn) {
			defer c.Close()
			target, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				return
			}
			defer target.Close()
			done := make(chan struct{}, 2)
			go func() { io.Copy(target, c); done <- struct{}{} }()
			go func() { io.Copy(c, target); done <- struct{}{} }()
			<-done
		}(conn)
	}
}

func extractTor() (string, func(), error) {
	dir, err := os.MkdirTemp("", fmt.Sprintf("nestor-tor-%d-*", os.Getpid()))
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }

	torBytes, err := decompress(torBinaryData)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to extract tor: %w", err)
	}
	torPath := filepath.Join(dir, torExeName)
	if err := os.WriteFile(torPath, torBytes, 0755); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := extractPlatformLibs(dir); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to extract platform libs: %w", err)
	}
	if err := signBinary(torPath); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to sign tor binary: %w", err)
	}
	return torPath, cleanup, nil
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	var port int
	var quiet bool
	flag.IntVar(&port, "port", 0, "Local port to forward through Tor (default: serve current directory)")
	flag.IntVar(&port, "p", 0, "Local port to forward (shorthand)")
	flag.BoolVar(&quiet, "quiet", false, "Only print the onion URL")
	flag.BoolVar(&quiet, "q", false, "Only print the onion URL (shorthand)")
	flag.Parse()

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
