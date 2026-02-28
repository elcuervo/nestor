//go:build !darwin

package main

func extractPlatformLibs(_ string) error {
	return nil
}

func signBinary(_ string) error {
	return nil
}
