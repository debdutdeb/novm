//go:build darwin

package utils

import (
	"os"

	"golang.org/x/sys/unix"
)

func IsInteractive() bool {
	_, err := unix.IoctlGetTermios(int(os.Stdout.Fd()), unix.TIOCGETA)
	return err == nil
}
