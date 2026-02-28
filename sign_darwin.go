//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func extractPlatformLibs(dir string) error {
	libBytes, err := decompress(torLibeventData)
	if err != nil {
		return fmt.Errorf("failed to extract libevent: %w", err)
	}
	libPath := filepath.Join(dir, "libevent-2.1.7.dylib")
	if err := os.WriteFile(libPath, libBytes, 0644); err != nil {
		return err
	}
	return exec.Command("codesign", "-s", "-", libPath).Run()
}

func signBinary(path string) error {
	return exec.Command("codesign", "-s", "-", path).Run()
}
