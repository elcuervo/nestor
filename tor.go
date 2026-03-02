package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cretz/bine/process"
	"github.com/cretz/bine/tor"
	"github.com/ulikunitz/xz"
)

type silentCreator struct {
	exePath string
	stderr  *bytes.Buffer
}
type silentProcess struct{ *exec.Cmd }

func (s *silentCreator) New(ctx context.Context, args ...string) (process.Process, error) {
	cmd := exec.CommandContext(ctx, s.exePath, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = s.stderr
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+filepath.Dir(s.exePath))
	return &silentProcess{cmd}, nil
}

func (s *silentProcess) EmbeddedControlConn() (net.Conn, error) {
	return nil, process.ErrControlConnUnsupported
}

func startTor(path, dataDir string) (*tor.Tor, error) {
	var stderr bytes.Buffer
	t, err := tor.Start(nil, &tor.StartConf{
		ProcessCreator: &silentCreator{exePath: path, stderr: &stderr},
		DebugWriter:    io.Discard,
		DataDir:        dataDir,
	})
	if err != nil && stderr.Len() > 0 {
		return nil, fmt.Errorf("%w\n%s", err, stderr.String())
	}
	return t, err
}

func extractTor() (torPath, dataDir string, cleanup func(), err error) {
	dir, err := os.MkdirTemp("", fmt.Sprintf("%s-%d-*", torTempDirPrefix, os.Getpid()))
	if err != nil {
		return "", "", nil, err
	}
	dataDir, err = os.MkdirTemp("", fmt.Sprintf("%s-%d-*", dataDirPrefix, os.Getpid()))
	if err != nil {
		os.RemoveAll(dir)
		return "", "", nil, err
	}
	cleanup = func() {
		os.RemoveAll(dir)
		os.RemoveAll(dataDir)
	}

	torBytes, err := decompress(torBinaryData)
	if err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("failed to extract tor: %w", err)
	}
	torPath = filepath.Join(dir, torExeName)
	if err := os.WriteFile(torPath, torBytes, 0755); err != nil {
		cleanup()
		return "", "", nil, err
	}
	if err := extractPlatformLibs(dir); err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("failed to extract platform libs: %w", err)
	}
	if err := signBinary(torPath); err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("failed to sign tor binary: %w", err)
	}
	return torPath, dataDir, cleanup, nil
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
				fmt.Fprintf(os.Stderr, "nestor: dial localhost:%d: %v\n", port, err)
				return
			}
			defer target.Close()
			done := make(chan struct{}, 2)
			go func() { io.Copy(target, c); done <- struct{}{} }()
			go func() { io.Copy(c, target); done <- struct{}{} }()
			<-done
			<-done
		}(conn)
	}
}

func isClosedNetErr(err error) bool {
	return errors.Is(err, net.ErrClosed)
}

func runQuiet(port int) {
	torPath, dataDir, cleanup, err := extractTor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	t, err := startTor(torPath, dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer t.Close()

	ctx, cancel := context.WithTimeout(context.Background(), onionCreateTimeout)
	defer cancel()

	onion, err := t.Listen(ctx, &tor.ListenConf{RemotePorts: []int{torRemotePort}, Version3: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer onion.Close()

	fmt.Printf("http://%v.onion\n", onion.ID)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		onion.Close()
	}()

	if port != 0 {
		proxyPort(onion, port)
	} else {
		http.Serve(onion, http.FileServer(http.Dir(".")))
	}
}
