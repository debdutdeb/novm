package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/debdutdeb/gopark/pkg/utils"
	"github.com/debdutdeb/novm/v3/common"
)

func TestUpdateNpmPrefix(t *testing.T) {
	sources := []string{
		"a=b\nc=d\nprefix=/usr/local/bin\n", // at the end with newline
		"a=b\nc=d\nprefix=/usr/local/bin",   // at the end without newline
		"prefix=/usr/local/bin\na=b",        // beginning
		"a=b\nc=d\n",                        // doesn't exist
	}

	myprefix := filepath.Join(os.Getenv("HOME"), common.NOVM_DIR)

	lineShouldContain := "prefix=" + myprefix
	lineShouldNotContain := "prefix=/usr/local/bin"

	for _, source := range sources {
		newConfig, err := updateNpmPrefix([]byte(source))
		if err != nil {
			t.Fatalf("got: %s\n, err: %v", newConfig, err)
		}

		if !strings.Contains(string(newConfig), lineShouldContain) {
			t.Fatalf("expected: %s, got: %s\n", lineShouldContain, newConfig)
		}

		if strings.Contains(string(newConfig), lineShouldNotContain) {
			t.Fatalf("expected not not contain: %s, got: %s\n", lineShouldNotContain, newConfig)
		}
	}
}

func TestLinks(t *testing.T) {
	files := []string{"node", "npm", "npx", "yarn"}

	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)

	for _, file := range files {
		dir, err := utils.MkdirTemp("", "testnovm")
		if err != nil {
			t.Error(err)
		}
		t.Logf("working dir: %s\n", dir)
		path := fmt.Sprintf("%s:%s", dir, oldPath)
		if err := os.Setenv("PATH", path); err != nil {
			t.Error(err)
		}

		realBin := filepath.Join(dir, file)
		if err := os.WriteFile(realBin, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatalf("failed to write fake binary %s: %v", realBin, err)
		}

		resolvedPath, err := exec.LookPath(file)
		if err != nil {
			t.Fatalf("lookpath for %s: %v", file, err)
		}

		target, err := resolveRealBin(resolvedPath)
		if err != nil {
			t.Fatalf("resolveRealBin: %v", err)
		}

		if target != realBin {
			t.Fatalf("expected resolved target %s, got %s", realBin, target)
		}

		if err := linkOthers(target, dir, files, file); err != nil {
			t.Fatalf("linkOthers: %v", err)
		}

		for _, other := range files {
			link := filepath.Join(dir, other)

			if other == file {
				if fi, err := os.Lstat(link); err == nil && fi.Mode()&os.ModeSymlink != 0 {
					t.Fatalf("%s should not have been symlinked to itself", other)
				}
				continue
			}

			got, err := os.Readlink(link)
			if err != nil {
				t.Fatalf("expected %s to be a symlink to %s: %v", other, target, err)
			}

			if got != target {
				t.Fatalf("expected %s -> %s, got -> %s", other, target, got)
			}
		}
	}
}
