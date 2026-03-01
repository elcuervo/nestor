//go:build !darwin && !linux

package main

func extractPlatformLibs(_ string) error {
	return nil
}

func signBinary(_ string) error {
	return nil
}
