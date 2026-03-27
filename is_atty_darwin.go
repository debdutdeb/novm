//go:build darwin

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func isInteractive() bool {
	_, err := unix.IoctlGetTermios(int(os.Stdout.Fd()), unix.TIOCGETA)
	return err == nil
}
