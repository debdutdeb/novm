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

	semverv3 "github.com/Masterminds/semver/v3"
	gopark "github.com/debdutdeb/gopark/pkg/utils"
)

var ErrNodeNotInstalled = errors.New("nodejs not installed")
var ErrNodeVersionNotFound = errors.New("nodejs version not found")

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
	versionStr  string
	arch        string
	rootDir     string
	installDir  string
	environment []string

	binPath string
	global  bool

	cache nCache

	version SemverManager
}

type NpmRunner interface {
	CaptureOutput(args ...string) ([]byte, []byte, error)
	Run(args ...string) error
}

func NewNodeManager(global bool, version string, rootDir string) (*N, error) {
	n := &N{
		global:     global,
		rootDir:    rootDir,
		versionStr: version,
	}

	var err error

	if err := n.initCache(); err != nil {
		return nil, err
	}

	switch version {
	case "latest":
		n.arch = n.getNodeJsArch()

		if n.versionStr, err = n.findLatestVersion(false); err != nil {
			return nil, err
		}
	case "lts":
		n.arch = n.getNodeJsArch()

		if n.versionStr, err = n.findLatestVersion(true); err != nil {
			return nil, err
		}
	default:
		if n.version, err = n.parseVersion(version); err != nil {
			return nil, err
		}

		n.arch = n.getNodeJsArch()

		found := false

		var releases []nCacheItem

		for _, release := range n.cache {
			c := n.version.Compare(semverv3.MustParse(release.Version))
			// if c == 3 {
			// 	break
			// }

			if c == 0 {
				releases = append(releases, release)

				break
			}

			if c == 2 {
				releases = append(releases, release)

				continue
			}
		}

		if len(releases) == 0 {
			return nil, fmt.Errorf("no release found for version: \"%s\"", version)
		}

		archiveType := n.getArchiveType()

	loop:
		for _, release := range releases {
			for _, thisType := range release.Files {
				if thisType == archiveType {
					n.versionStr = release.Version

					found = true

					break loop
				}
			}
		}

		if !found {
			return nil, fmt.Errorf("version %s not found for file %s, err: %w", version, archiveType, ErrNodeVersionNotFound)
		}
	}

	n.installDir = filepath.Join(rootDir, "versions", n.versionStr, runtime.GOOS, n.arch)
	n.environment = append(os.Environ(), "NP_NODE_VERSION="+n.versionStr) // make sure we continue using this version on every nested call (like lifecycle scripts) in case source isn't environment variable

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

func (n *N) parseVersion(version string) (SemverManager, error) {
	var semverManager SemverManager

	semverManager, err1 := semverv3.NewVersion(version)
	if err1 == nil {
		return semverManager, nil
	}

	c, err2 := semverv3.NewConstraint(version)
	if err2 == nil {
		return semverv3Constraints(*c), nil
	}

	return nil, fmt.Errorf("failed to parse version, neither a semver nor constraint: %w, %w", err1, err2)
}

func (n *N) getNodeJsArch() string {
	if runtime.GOARCH == "amd64" {
		return "x64"
	}

	if n.versionStr == "latest" || n.versionStr == "lts" {
		return runtime.GOARCH
	}

	// TODO: try to remove these type assertions
	if runtime.GOOS == "darwin" {
		c, _ := semverv3.NewConstraint("<16.0.0")

		// if source has a constraint set, unfortunately for now
		// get the actual typed variable out and use that
		constraint, ok := n.version.(semverv3Constraints)
		if ok {
			if semverv3.Constraints(constraint).CheckConstraints(c) {
				return "x64"
			}

			return runtime.GOARCH
		}

		if c.Check(n.version.(*semverv3.Version)) {
			return "x64"
		}
	}

	return runtime.GOARCH
}

// SetBinaryArchX86 is used for apple m-series/arm series machines
// v < 16 doesn't have arm binaries
func (n *N) SetBinaryArchX86() {
	n.arch = "x64"
}

func (n *N) getArchiveType() string {
	finalArch := n.arch

	if runtime.GOOS == "linux" {
		return "linux-" + finalArch
	}

	if runtime.GOOS == "darwin" {
		return "osx-" + finalArch + "-tar"
	}

	return runtime.GOOS + "-" + runtime.GOARCH
}

func (n *N) findLatestVersion(lts bool) (string, error) {
	fileType := n.getArchiveType()

	isLts := func(release *nCacheItem) bool {
		if _, ok := release.Lts.(string); !ok {
			return false
		}

		return true
	}

	for _, release := range n.cache {
		if lts && !isLts(&release) {
			continue
		}

		for _, file := range release.Files {
			if file == fileType {
				return release.Version, nil
			}
		}
	}

	return "", fmt.Errorf("failed to find latest version for file %s", fileType)
}

func (n *N) Npm() NpmRunner {
	npm := *n
	npm.binPath = filepath.Join(filepath.Dir(n.binPath), "npm")
	return &npm
}

func (n *N) Install() error {
	tmpDir, err := gopark.MkdirTemp("", "novm")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory to install nodejs: %v", err)
	}

	url, filename := n._assets()

	archivePath := filepath.Join(tmpDir, filename)

	err = gopark.DownloadWithProgressBar("Node "+n.versionStr, url, archivePath)
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
	if n.versionStr == n.Version() {
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
	filename = fmt.Sprintf("node-%s-%s-%s.tar.xz", n.versionStr, runtime.GOOS, n.arch)
	url = fmt.Sprintf("https://nodejs.org/download/release/%s/%s", n.versionStr, filename)
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
