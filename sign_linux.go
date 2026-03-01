//go:build linux

package main

import (
	"os"
	"path/filepath"
)

func extractPlatformLibs(dir string) error {
	libBytes, err := decompress(torLibeventData)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "libevent-2.1.so.7"), libBytes, 0644)
}

func signBinary(_ string) error {
	return nil
}
