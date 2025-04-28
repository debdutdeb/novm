package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/debdutdeb/node-proxy/common"
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
