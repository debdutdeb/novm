package main

/*
does some new install stuff
*/

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/debdutdeb/node-proxy/common"
	"github.com/debdutdeb/node-proxy/state"
)

func handleNewInstall() error {
	s, err := state.NewState()
	if err != nil {
		return err
	}

	if !s.Update.LastChecked.Equal(time.Time{}) {
		return nil
	}

	// never checked for update == first install, i.o.w state is empty

	if err := setnpmPrefix(); err != nil {
		return err
	}

	me := os.Args[0]

	switch bin := filepath.Base(me); bin {
	case "node":
		return os.Symlink(me, filepath.Join(filepath.Dir(me), "npm"))
	case "npm":
		return os.Symlink(me, filepath.Join(filepath.Dir(me), "node"))
	default:
		return fmt.Errorf("what binary is this? I should either be node or npm, but seems like I am %s", bin)
	}
}

func setnpmPrefix() error {
	f, err := os.OpenFile(filepath.Join(os.Getenv("HOME"), ".npmrc"), os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		return err
	}

	// TODO(me): optimize maybe
	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	lines := strings.Split(string(b), "\n")

	for i, line := range lines {
		if len(line) >= 7 && line[:7] == "prefix=" {
			lines[i] = "prefix=" + filepath.Join(os.Getenv("HOME"), common.NOVM_DIR)
			break
		}
	}

	f.Seek(0, io.SeekStart)

	_, err = f.WriteString(strings.Join(lines, "\n"))

	return err
}
