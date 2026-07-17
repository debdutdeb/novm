package common

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
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

func DepthSourceDetection() int {
	var depth = os.Getenv("NOVM_DEPTH_SOURCE_DETECTION")
	if depth == "" {
		return 2
	}
	if n, err := strconv.Atoi(depth); err != nil {
		return 2
	} else {
		return n
	}
}
