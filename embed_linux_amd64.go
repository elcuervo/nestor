//go:build linux && amd64

package main

import _ "embed"

//go:embed tor_binaries/linux-amd64/tor.xz
var torBinaryData []byte

const torExeName = "tor"
