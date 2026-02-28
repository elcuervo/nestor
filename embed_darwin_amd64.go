//go:build darwin && amd64

package main

import _ "embed"

//go:embed tor_binaries/darwin-amd64/tor.xz
var torBinaryData []byte

//go:embed tor_binaries/darwin-amd64/libevent-2.1.7.dylib.xz
var torLibeventData []byte

const torExeName = "tor"
