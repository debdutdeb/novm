package common

import (
	"log"
	"os/user"
	"path/filepath"
)

const NOVM_DIR = ".novm"

const BIN_NAME = "novm"

func rootDir() func() (string, error) {
	u, err := user.Current()
	if err != nil {
		log.Fatalf("failed to detect current user: %w", err)
	}

	root := filepath.Join(u.HomeDir, NOVM_DIR)

	return func() (string, error) {
		return root, err
	}
}

var RootDir = rootDir()
