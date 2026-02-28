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
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/cretz/bine/tor"
	"github.com/fatih/color"
	"github.com/ulikunitz/xz"
)

var (
	purple = color.New(color.FgMagenta, color.Bold)
	green  = color.New(color.FgGreen, color.Bold)
	cyan   = color.New(color.FgCyan)
	dim    = color.New(color.Faint)
	bold   = color.New(color.Bold)
	red    = color.New(color.FgRed, color.Bold)
)

const logoText = `
                  _
  _ __   ___  ___| |_ ___  _ __
 | '_ \ / _ \/ __| __/ _ \| '__|
 | | | |  __/\__ \ || (_) | |
 |_| |_|\___||___/\__\___/|_|
     NEtwork Share via TOR`

func urlBox(url string) string {
	bar := strings.Repeat("─", len(url)+4)
	return fmt.Sprintf("\n  ┌%s┐\n  │  %s  │\n  └%s┘", bar, bold.Sprint(url), bar)
}

func runPhase(s *spinner.Spinner, spinMsg, doneMsg string, fn func() error) error {
	s.Suffix = "  " + spinMsg
	s.Start()
	err := fn()
	if err != nil {
		s.FinalMSG = red.Sprint("  ✗ ") + spinMsg + "\n"
	} else {
		s.FinalMSG = green.Sprint("  ✓ ") + doneMsg + "\n"
	}
	s.Stop()
	return err
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

func main() {
	var port int
	flag.IntVar(&port, "port", 0, "Local port to forward through Tor (default: serve current directory)")
	flag.IntVar(&port, "p", 0, "Local port to forward (shorthand)")
	flag.Parse()

	purple.Print(logoText)
	fmt.Print("\n\n")

	s := spinner.New(spinner.CharSets[11], 80*time.Millisecond)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var torPath string
	var cleanup func()
	if err := runPhase(s, "Extracting Tor binary...", "Tor binary extracted", func() error {
		var err error
		torPath, cleanup, err = extractTor()
		return err
	}); err != nil {
		red.Fprintf(os.Stderr, "  Error: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	go func() {
		<-c
		cleanup()
		os.Exit(0)
	}()

	var t *tor.Tor
	if err := runPhase(s, "Starting Tor process...", "Tor process started", func() error {
		var err error
		t, err = tor.Start(nil, &tor.StartConf{ExePath: torPath})
		return err
	}); err != nil {
		red.Fprintf(os.Stderr, "  Error: %v\n", err)
		os.Exit(1)
	}
	defer t.Close()

	listenCtx, listenCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer listenCancel()

	var onion *tor.OnionService
	if err := runPhase(s, "Creating onion service...", "Onion service ready", func() error {
		var err error
		onion, err = t.Listen(listenCtx, &tor.ListenConf{RemotePorts: []int{80}, Version3: true})
		return err
	}); err != nil {
		red.Fprintf(os.Stderr, "  Error: %v\n", err)
		os.Exit(1)
	}
	defer onion.Close()

	url := fmt.Sprintf("http://%v.onion", onion.ID)
	fmt.Println(urlBox(url))

	if port != 0 {
		fmt.Printf("\n  Forwarding ")
		cyan.Printf("localhost:%d", port)
		fmt.Println(" → Tor")
	} else {
		dir, _ := os.Getwd()
		fmt.Printf("\n  Serving ")
		cyan.Println(dir)
	}
	fmt.Println()
	dim.Println("  Press Ctrl+C to stop")
	fmt.Println()

	errCh := make(chan error, 1)
	if port != 0 {
		go func() { errCh <- proxyPort(onion, port) }()
	} else {
		go func() { errCh <- http.Serve(onion, http.FileServer(http.Dir("."))) }()
	}

	if err := <-errCh; err != nil {
		red.Fprintf(os.Stderr, "\n  ✗ Failed serving: %v\n", err)
		os.Exit(1)
	}
}
