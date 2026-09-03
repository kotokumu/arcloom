//go:build !linux && !darwin

package main

import "os"

// The reference command's FIFO contract is supported on Linux and macOS.
func main() { os.Exit(1) }
