//go:build linux

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func isInteractive() bool {
	_, err := unix.IoctlGetTermios(int(os.Stdout.Fd()), unix.TCGETS)
	return err == nil
}
