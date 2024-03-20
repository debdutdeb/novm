package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

func installCommand(rootDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use: "install",
		Run: func(c *cobra.Command, args []string) {
			if err := install(filepath.Join(rootDir, "bin")); err != nil {
				log.Fatalf("failed to install novm as node: %v", err)
			}
		},
	}

	return cmd
}

func findMaxInstalledVersion(rootDir string) (string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "latest", nil
		}

		return "", err
	}

	if len(entries) == 0 {
		return "latest", nil
	}

	if !semver.IsValid(entries[0].Name()) {
		return "", fmt.Errorf("root install directory seems to be polluted with files unknown %s", entries[0].Name())
	}

	max := entries[0].Name()

	for _, entry := range entries[1:] {
		if !semver.IsValid(entry.Name()) {
			return "", fmt.Errorf("root install directory seems to be polluted with files unknown %s", entry.Name())
		}

		if semver.Compare(entry.Name(), max) == 1 {
			max = entry.Name()
		}
	}

	return max, nil
}

func install(rootDir string) error {
	if err := os.MkdirAll(rootDir, 0750); err != nil {
		return err
	}

	_, err := exec.LookPath("node")
	if err == nil {
		log.Printf("you will need to add %s to your shell's rc file as you already have nodejs installed", rootDir)
	}

	binPath, err := exec.LookPath(os.Args[0])
	if err != nil {
		return err
	}

	if err = forceSymlink(binPath, filepath.Join(rootDir, "node")); err != nil {
		return err
	}

	if err = forceSymlink(binPath, filepath.Join(rootDir, "npm")); err != nil {
		return err
	}

	return setnpmPrefix()
}

func setnpmPrefix() error {
	f, err := os.OpenFile(filepath.Join(os.Getenv("HOME"), ".npmrc"), os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		return err
	}

	// TODO(me): optimize
	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	lines := strings.Split(string(b), "\n")

	for i, line := range lines {
		if len(line) >= 7 && line[:7] == "prefix=" {
			lines[i] = "prefix=" + filepath.Join(os.Getenv("HOME"), ".node-proxy")
			break
		}
	}

	f.Seek(0, io.SeekStart)

	_, err = f.WriteString(strings.Join(lines, "\n"))

	return err
}

func forceSymlink(oldname, newname string) error {
	if err := os.Symlink(oldname, newname); err != nil {
		if errors.Is(err, os.ErrExist) {
			err = os.Remove(newname)
			if err != nil {
				return err
			}

			return os.Symlink(oldname, newname)
		}

		return err
	}

	return nil
}
