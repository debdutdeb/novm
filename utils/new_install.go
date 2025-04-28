package utils

/*
does some new install stuff
*/

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/debdutdeb/node-proxy/common"
	"github.com/debdutdeb/node-proxy/state"
)

var errNotWriteable = errors.New("does not have permission to write to dir")

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

	if err := linkFiles(me, filepath.Join(dir, linkTo)); err != nil {
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

	replaced := false

	// what will eventually be written
	bytes := make([]byte, len(b))

	// lines, basically
	newlines := []int{}

	prefixReg := regexp.MustCompile("^prefix=")

	for i, b_ := range b {
		bytes[i] = b_
		if b_ != '\n' {
			newlines = append(newlines, i)
			continue
		}

		if prefixReg.Match(bytes) {
			replaced = true
			prefixBytes := []byte("prefix=" + prefix + "\n")
			bytes = bytes[:newlines[len(newlines)-2]] // skip this line
			bytes = append(bytes, append(prefixBytes, b[i:]...)...)
			break
		}
	}

	if !replaced {
		bytes = append(bytes, []byte("prefix="+prefix+"\n")...)
	}

	f.Seek(0, io.SeekStart)

	_, err = f.Write(bytes)

	return err
}

func linkFiles(path1, path2 string) error {
	// ????
	f1, err := os.Stat(path1)
	if err != nil {
		return fmt.Errorf("unable to access file %s, err: %v", path1, err)
	}

	if !f1.IsDir() {
		return fmt.Errorf("%s is not a file", path1)
	}

	err = isWritable(filepath.Dir(path2))

	if err != nil {
		if errors.Is(err, errNotWriteable) {
			return exec.Command("sudo", "ln", "-s", path1, path2).Run()
		}

		return fmt.Errorf("failed to link files, err: %w", err)
	}

	return os.Symlink(path1, path2)
}

func isWritable(dir string) error {
	// expect dir
	s, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !s.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	me, err := user.Current()
	if err != nil {
		return err
	}

	uid, _ := strconv.Atoi(me.Uid)

	if uid == 0 {
		return errors.New("") // run as root no need for sudo
	}

	gid, _ := strconv.Atoi(me.Gid)

	perms := s.Mode().Perm()

	if perms&0002 == 0002 { // other writeable, ownership doesn't matter
		return nil
	}

	sysStat, ok := s.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("unable to detect dir ownershit for %s", dir)
	}

	if sysStat.Uid == uint32(uid) && perms&0200 == 0200 {
		return nil
	}

	if sysStat.Gid == uint32(gid) && perms&0020 == 0020 {
		return nil
	}

	return errNotWriteable
}
