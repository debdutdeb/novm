package common

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
)

var RootDir string

func init() {
	workdir := os.Getenv("NOVM_WORKDIR")
	if workdir != "" {
		RootDir = workdir
		return
	}

	root, err := rootDir()()
	if err != nil {
		log.Fatalf("failed to detect current user: %v", err)
	}

	RootDir = root
}

const NOVM_DIR = ".novm"

const BIN_NAME = "novm"

func rootDir() func() (string, error) {
	u, err := user.Current()
	if err != nil {
		log.Fatalf("failed to detect current user: %v", err)
	}

	root := filepath.Join(u.HomeDir, NOVM_DIR)

	return func() (string, error) {
		return root, err
	}
}
