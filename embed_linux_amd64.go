//go:build linux && amd64

package main

import _ "embed"

//go:embed tor_binaries/linux-amd64/tor.xz
var torBinaryData []byte

//go:embed tor_binaries/linux-amd64/libevent-2.1.so.7.xz
var torLibeventData []byte

const torExeName = "tor"
