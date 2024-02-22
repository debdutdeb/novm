package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gopark "github.com/debdutdeb/gopark/pkg/utils"
	"golang.org/x/mod/semver"
)

var ErrNodeNotInstalled = errors.New("nodejs not installed")

// the node version manager

type nCacheItem struct {
	Version  string      `json:"version"`
	Date     string      `json:"date"`
	Files    []string    `json:"files"`
	Lts      interface{} `json:"lts,omitempty"`
	Security bool        `json:"security"`
}

type nCache []nCacheItem

type N struct {
	version     string
	arch        string
	rootDir     string
	installDir  string
	environment []string

	binPath string
	global  bool

	cache nCache
}

type NpmRunner interface {
	CaptureOutput(args ...string) ([]byte, []byte, error)
	Run(args ...string) error
}

func NewNodeManager(global bool, version string, rootDir string) (*N, error) {
	n := &N{
		global:  global,
		version: version,
		rootDir: rootDir,
	}

	if err := n.initCache(); err != nil {
		return nil, err
	}

	if runtime.GOARCH == "amd64" ||
		(runtime.GOOS == "darwin" &&
			semver.IsValid(n.version) &&
			semver.Compare(n.version, "v16.0.0") == -1) {
		n.arch = "x64"
	} else {
		// TODO is this right? given how x64 differs
		n.arch = runtime.GOARCH
	}

	// filetype is what kind of download artifact I am looking for
	myFileType := ""

	if runtime.GOOS == "darwin" {
		myFileType = "osx-" + n.arch + "-tar"
	} else {
		myFileType = runtime.GOOS + "-" + runtime.GOARCH
	}

	switch version {
	case "latest":
	latest_release:
		for _, release := range n.cache {
			for _, fileType := range release.Files {
				if fileType == myFileType {
					n.version = release.Version
					break latest_release
				}
			}
		}
	case "lts":
	lts_release:
		for _, release := range n.cache {
			_, ok := release.Lts.(string)
			if !ok {
				continue
			}

			for _, fileType := range release.Files {
				if fileType == myFileType {
					n.version = release.Version
					break lts_release
				}
			}
		}
	default:
		if version[0] != 'v' {
			version = "v" + version
		}

		found := false

		for _, release := range n.cache {
			if release.Version != version {
				continue
			}

			found = true

			for _, fileType := range release.Files {
				if fileType == myFileType {
					n.version = release.Version
				}
			}

			if found && n.version == "" {
				return nil, fmt.Errorf("given version not found for your current platform")
			} else if !found {
				return nil, fmt.Errorf("invalid nodejs version")
			}

			break
		}
	}

	if n.version == "" {
		return nil, fmt.Errorf("failed to find the latest release for your platform")
	}

	n.installDir = filepath.Join(rootDir, "versions", n.version, runtime.GOOS, n.arch)
	n.environment = append(os.Environ(), "NP_NODE_VERSION="+n.version) // make sure we continue using this version on every nested call (like lifecycle scripts) in case source isn't environment variable

	if n.global {
		binPath, err := exec.LookPath("node")
		if err != nil && errors.Is(err, exec.ErrNotFound) {
			return nil, ErrNodeNotInstalled
		} else if err != nil {
			return nil, fmt.Errorf("unknown error trying to detect nodejs global installation: %v", err)
		}

		n.binPath = binPath

		return n, nil
	}

	n.binPath = filepath.Join(n.installDir, "bin", "node")

	path := os.Getenv("PATH")

	n.environment = append([]string{fmt.Sprintf("PATH=%s:%s", filepath.Dir(n.binPath), path)}, n.environment...)

	return n, nil
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

func (n *N) Run(args ...string) (err error) {
	cmd := exec.Command(n.binPath, args...)

	// let the command take over
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = n.environment

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
	filename = fmt.Sprintf("node-%s-%s-%s.tar.xz", n.version, runtime.GOOS, n.arch)
	url = fmt.Sprintf("https://nodejs.org/download/release/%s/%s", n.version, filename)
	return
}

func (n *N) initCache() error {
	if stat, err := os.Stat(n.rootDir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(n.rootDir, 0750)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if !stat.IsDir() {
			return errors.New("can not continue since expected root directory is not a directory")
		}
	}

	cacheFilename := filepath.Join(n.rootDir, "node_versions.json")

	var (
		cacheExists bool = true
		stat        fs.FileInfo
		err         error
	)

	if stat, err = os.Stat(cacheFilename); err != nil {
		if os.IsNotExist(err) {
			cacheExists = false
		} else {
			return err
		}
	}

	cacheFile, err := os.OpenFile(cacheFilename, os.O_CREATE|os.O_RDWR, 0750)
	if err != nil {
		return err
	}

	var data nCache

	if cacheExists && time.Since(stat.ModTime()) < (time.Hour*24) {
		if err = json.NewDecoder(cacheFile).Decode(&data); err != nil {
			return err
		}

		n.cache = data
		return nil
	}

	resp, err := http.Get("https://nodejs.org/download/release/index.json")
	if err != nil {
		return err
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	resp.Body.Close()

	_, err = cacheFile.Write(content)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(content, &data); err != nil {
		return err
	}

	n.cache = data

	return nil
}
