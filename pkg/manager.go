package pkg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	gopark "github.com/debdutdeb/gopark/pkg/utils"
)

var ErrNodeNotInstalled = errors.New("nodejs not installed")

// the node version manager

type N struct {
	version     string
	arch        string
	installDir  string
	environment []string

	binPath string
	global  bool
}

type NpmRunner interface {
	CaptureOutput(args ...string) ([]byte, []byte, error)
}

func NewNodeManager(global bool, version string, rootDir string) (*N, error) {

	if version[0] != 'v' {
		version = "v" + version
	}

	n := &N{
		global:  global,
		version: version,
		arch:    runtime.GOARCH,
	}

	n.installDir = filepath.Join(rootDir, version, runtime.GOOS, runtime.GOARCH)

	n.environment = os.Environ()

	return n, n.findInstall()
}

func (n *N) Npm() NpmRunner {
	npm := *n
	npm.binPath = filepath.Join(filepath.Dir(n.binPath), "npm")
	return &npm
}

func (n *N) Install() error {
	tmpDir, err := gopark.MkdirTemp()
	if err != nil {
		return fmt.Errorf("failed to create temporary directory to install nodejs: %v", err)
	}

	url, filename := n._assets()

	archivePath := filepath.Join(tmpDir, filename)

	err = gopark.DownloadWithProgressBar("Node "+n.version, url, archivePath)
	if err != nil {
		return err
	}

	cmd := exec.Command("xz", "--decompress", archivePath)
	err = cmd.Run()
	if err != nil {
		return err
	}

	archivePath = strings.TrimSuffix(archivePath, ".xz")

	cmd = exec.Command("tar", "xf", archivePath)
	cmd.Dir = tmpDir
	if err = cmd.Run(); err != nil {
		return err
	}

	archivePath = strings.TrimSuffix(archivePath, ".tar")

	toInstall := []string{"share", "lib", "include", "bin"}

	var dst = n.installDir

	if n.global {
		dst = "/usr/local"
	}

	for _, loc := range toInstall {
		if err := gopark.DumbInstall(filepath.Join(dst, loc), filepath.Join(archivePath, loc)); err != nil {
			return err
		}
	}

	return nil
}

func (n *N) EnsureInstalled() error {
	if n.version == n.Version() {
		return nil
	}

	return n.Install()
}

func (n *N) findInstall() error {
	if n.global {
		binPath, err := exec.LookPath("node")
		if err != nil && errors.Is(err, exec.ErrNotFound) {
			return ErrNodeNotInstalled
		} else if err != nil {
			return fmt.Errorf("unknown error trying to detect nodejs global installation: %v", err)
		}

		n.binPath = binPath

		return nil
	}

	n.binPath = filepath.Join(n.installDir, "bin/node")

	path := os.Getenv("PATH")

	n.environment = append([]string{fmt.Sprintf("PATH=%s:%s", filepath.Dir(n.binPath), path)}, n.environment...)

	return nil
}

func (n *N) Run(args ...string) (err error) {
	cmd := exec.Command(n.binPath, args...)

	// let the command take over
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return
	}

	err = cmd.Wait()

	return
}

// Deprecated
func (n *N) CaptureOutput(args ...string) (stdout []byte, stderr []byte, err error) {
	cmd := exec.Command(n.binPath, args...)

	cmd.Env = n.environment

	stdout, err = cmd.Output()
	if err != nil {
		e, _ := err.(*exec.ExitError)
		if e == nil {
			return
		}
		stderr = e.Stderr
		return
	}

	return
}

func (n *N) Version() string {
	out, _, _ := n.CaptureOutput("--version")
	if len(out) == 0 {
		return string(out)
	}
	return string(out[:len(out)-1])
}

func (n *N) _assets() (url string, filename string) {
	filename = fmt.Sprintf("node-%s-%s-%s.tar.xz", n.version, runtime.GOOS, runtime.GOARCH)
	url = fmt.Sprintf("https://nodejs.org/download/release/%s/%s", n.version, filename)
	return
}
