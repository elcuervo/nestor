package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/cretz/bine/tor"
	"github.com/ulikunitz/xz"
)

const logo = `
                  _
  _ __   ___  ___| |_ ___  _ __
 | '_ \ / _ \/ __| __/ _ \| '__|
 | | | |  __/\__ \ || (_) | |
 |_| |_|\___||___/\__\___/|_|
     NEtwork Share via TOR
`

func decompress(data []byte) ([]byte, error) {
	r, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
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
	fmt.Println(logo)

	s := spinner.New(spinner.CharSets[4], 100*time.Millisecond)
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	s.Suffix = " Finding an available tor address."
	s.Start()

	torPath, cleanup, err := extractTor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to extract Tor: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	go func() {
		<-c
		cleanup()
		os.Exit(0)
	}()

	t, err := tor.Start(nil, &tor.StartConf{ExePath: torPath})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start Tor: %v\n", err)
		os.Exit(1)
	}

	defer t.Close()

	listenCtx, listenCancel := context.WithTimeout(context.Background(), 3*time.Minute)

	defer listenCancel()

	onion, err := t.Listen(listenCtx, &tor.ListenConf{RemotePorts: []int{80}, Version3: true})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create onion service: %v\n", err)
		os.Exit(1)
	}

	defer onion.Close()

	s.Stop()

	fmt.Printf("Go to http://%v.onion\n", onion.ID)

	errCh := make(chan error, 1)

	go func() { errCh <- http.Serve(onion, http.FileServer(http.Dir("."))) }()

	if err = <-errCh; err != nil {
		fmt.Fprintf(os.Stderr, "Failed serving: %v\n", err)
		os.Exit(1)
	}
}
