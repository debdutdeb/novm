package utils

/*
does some new install stuff
*/

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/debdutdeb/node-proxy/common"
	"github.com/debdutdeb/node-proxy/state"
)

func HandleNewInstall() error {
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

	bin := filepath.Base(me)

	var dir string

	path, err := exec.LookPath(me) // for "path", this will always error
	if err == nil {
		dir = filepath.Dir(path)
	} else {
		dir = filepath.Dir(me)
	}

	var linkTo string
	switch bin {
	case "node":
		linkTo = filepath.Join(dir, "npm")
	case "npm":
		linkTo = filepath.Join(dir, "node")
	default:
		return fmt.Errorf("what binary is this? I should either be node or npm, but seems like I am %s", bin)
	}

	fmt.Printf("Linking %s to %s\n", me, linkTo)

	if err := sudoLn(me, filepath.Join(dir, linkTo)); err != nil {
		return err
	}

	return nil
}

func setnpmPrefix() error {
	prefix := filepath.Join(os.Getenv("HOME"), common.NOVM_DIR)

	f, err := os.OpenFile(filepath.Join(os.Getenv("HOME"), ".npmrc"), os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		return err
	}

	// TODO(me): optimize maybe
	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	if len(b) == 0 {
		_, err := f.WriteString(fmt.Sprintf("prefix=%s\n", prefix))
		return err
	}

	lines := strings.Split(string(b), "\n")

	replaced := false

	for i, line := range lines {
		if len(line) >= 7 && line[:7] == "prefix=" {
			lines[i] = "prefix=" + prefix
			replaced = true
			break
		}
	}

	if !replaced {
		lines = append(lines, "prefix="+prefix)
	}

	f.Seek(0, io.SeekStart)

	_, err = f.WriteString(strings.Join(lines, "\n"))

	return err
}

func sudoLn(path1, path2 string) error {
	// ????
	f1, err := os.Stat(path1)
	if err != nil {
		return fmt.Errorf("unable to access file %s, err: %v", path1, err)
	}

	if !f1.IsDir() {
		return fmt.Errorf("%s is not a file", path1)
	}

	f2, err := os.Stat(path2)
	if err != nil {
		return fmt.Errorf("unable to access file %s, err: %v", path2, err)
	}

	if !f2.IsDir() {
		return fmt.Errorf("%s is not a file", path2)
	}

	return exec.Command("sudo", "ln", "-s", path1, path2).Run()
}
