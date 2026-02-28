//go:build darwin && arm64

package main

import _ "embed"

//go:embed tor_binaries/darwin-arm64/tor.xz
var torBinaryData []byte

//go:embed tor_binaries/darwin-arm64/libevent-2.1.7.dylib.xz
var torLibeventData []byte

const torExeName = "tor"
