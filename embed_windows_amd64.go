//go:build windows && amd64

package main

import _ "embed"

//go:embed tor_binaries/windows-amd64/tor.exe.xz
var torBinaryData []byte

const torExeName = "tor.exe"
