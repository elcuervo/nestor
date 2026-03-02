package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

// makeXZ encodes content with xz compression for use in decompress tests.
func makeXZ(t *testing.T, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := xz.NewWriter(&buf)
	require.NoError(t, err)
	_, err = w.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestDecompress(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		original := []byte("hello nestor")
		got, err := decompress(makeXZ(t, original))
		require.NoError(t, err)
		assert.Equal(t, original, got)
	})

	t.Run("empty input returns error", func(t *testing.T) {
		_, err := decompress([]byte{})
		assert.Error(t, err)
	})

	t.Run("garbage input returns error", func(t *testing.T) {
		_, err := decompress([]byte{0xde, 0xad, 0xbe, 0xef})
		assert.Error(t, err)
	})

	t.Run("large input roundtrips", func(t *testing.T) {
		original := bytes.Repeat([]byte("a"), 1<<20)
		got, err := decompress(makeXZ(t, original))
		require.NoError(t, err)
		assert.Equal(t, original, got)
	})
}

func TestIsClosedNetErr(t *testing.T) {
	t.Run("nil returns false", func(t *testing.T) {
		assert.False(t, isClosedNetErr(nil))
	})

	t.Run("net.ErrClosed returns true", func(t *testing.T) {
		assert.True(t, isClosedNetErr(net.ErrClosed))
	})

	t.Run("wrapped net.ErrClosed returns true", func(t *testing.T) {
		assert.True(t, isClosedNetErr(fmt.Errorf("wrapped: %w", net.ErrClosed)))
	})

	t.Run("io.EOF returns false", func(t *testing.T) {
		assert.False(t, isClosedNetErr(io.EOF))
	})

	t.Run("plain string match returns false", func(t *testing.T) {
		// Guards against regression to string-matching behaviour.
		assert.False(t, isClosedNetErr(fmt.Errorf("use of closed network connection")))
	})
}

func TestProxyPort(t *testing.T) {
	t.Run("forwards data bidirectionally", func(t *testing.T) {
		// Backend: simple echo server
		backend, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer backend.Close()
		backendPort := backend.Addr().(*net.TCPAddr).Port

		go func() {
			conn, err := backend.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
			io.Copy(conn, conn) // echo
		}()

		// Onion-side listener that proxyPort will accept from.
		onionSide, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer onionSide.Close()

		go proxyPort(onionSide, backendPort) //nolint:errcheck

		// Client connects to the onion-side listener.
		client, err := net.Dial("tcp", onionSide.Addr().String())
		require.NoError(t, err)
		defer client.Close()

		msg := []byte("ping")
		_, err = client.Write(msg)
		require.NoError(t, err)

		buf := make([]byte, len(msg))
		_, err = io.ReadFull(client, buf)
		require.NoError(t, err)
		assert.Equal(t, msg, buf)
	})

	t.Run("closed listener returns error immediately", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		l.Close()

		err = proxyPort(l, 9999)
		assert.Error(t, err)
	})

	t.Run("dial failure does not block accept loop", func(t *testing.T) {
		// Port with nothing listening — dial will fail silently but Accept loop continues.
		onionSide, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		// Use a port that should have nothing listening on it.
		// We don't care about the proxyPort return value here; just that it doesn't
		// block. We close the listener to terminate it after a connection attempt.
		go func() {
			// Connect then immediately close — triggers the dial-failure path.
			c, err := net.Dial("tcp", onionSide.Addr().String())
			if err == nil {
				c.Close()
			}
			onionSide.Close()
		}()

		err = proxyPort(onionSide, 1) // port 1: almost certainly not listening
		assert.Error(t, err)
	})
}
