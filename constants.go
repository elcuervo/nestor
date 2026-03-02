package main

import "time"

const (
	torRemotePort      = 80
	onionCreateTimeout = 3 * time.Minute
	torTempDirPrefix   = "nestor-tor"
	dataDirPrefix      = "nestor-data"
	minPort            = 1
	maxPort            = 65535
)
