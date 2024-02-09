package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/debdutdeb/node-proxy/pkg"
	"golang.org/x/mod/semver"
)

type packageJson struct {
	Engines struct {
		Node string `json:"node"`
	} `json:"engines"`
}

var NodeJsVersion string = ""

func init() {
	if v := os.Getenv("NP_NODE_VERSION"); v != "" {
		NodeJsVersion = v
		return
	}

	if s, err := os.Stat("package.json"); err == nil && !s.IsDir() {
		var p packageJson
		f, err := os.Open("package.json")
		if err != nil {
			log.Fatalf("failed to open package.json: %v", err)
		}

		defer f.Close()

		err = json.NewDecoder(f).Decode(&p)
		if err != nil {
			log.Fatalf("failed to read package.json: %v", err)
		}

		NodeJsVersion = p.Engines.Node

		return
	}

	if s, err := os.Stat(".nvmrc"); err == nil && !s.IsDir() {
		// only supports the version
		f, err := os.Open(".nvmrc")
		if err != nil {
			log.Fatalf("failed to read .nvmrc: %v", err)
		}

		defer f.Close()

		b, err := io.ReadAll(f)
		if err != nil {
			log.Fatalf("failed to read .nvmrc: %v", err)
		}

		if b[len(b)-1] == 10 {
			b = b[:len(b)-1]
		}

		NodeJsVersion = string(b)
	}
}

func main() {
	var err error

	u, err := user.Current()
	if err != nil {
		log.Fatalf("failed to detect current user: %v", err)
	}

	root := filepath.Join(u.HomeDir, ".node-proxy")

	if filepath.Base(os.Args[0]) == "node-proxy" {
		log.Println("Running node-proxy without passing to node binary")
		switch os.Args[1] {
		case "install":
			if err = install(filepath.Join(root, "bin")); err != nil {
				log.Fatalf("failed to initialise node proxy: %v", err)
			}
		default:
			log.Println("unknown command")
		}
		return
	}

	root = filepath.Join(root, "versions")

	if NodeJsVersion == "" {
		log.Println("no nodejs version detected from sources, using latest installed")
		NodeJsVersion, err = findMaxInstalledVersion(root)
		if err != nil {
			log.Fatalf("failed to detect current nodejs version: %v", err)
		}
	}
	n, err := pkg.NewNodeManager(false, NodeJsVersion, root)
	if err != nil {
		panic(fmt.Errorf("failed to initialize node manager: %v", err))
	}

	err = n.EnsureInstalled()
	if err != nil {
		log.Fatalf("failed to install node version %v", err)
	}

	if os.Args[0] == "npm" {
		if err = n.Npm().Run(os.Args[1:]...); err != nil {
			log.Fatalf("failed to run npm: %v", err)
		}

		return
	}

	err = n.Run(os.Args[1:]...)
	if err != nil {
		log.Fatalf("failed to run nodejs: %v", err)
	}
}

func findMaxInstalledVersion(rootDir string) (string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return "", err
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

	bin1, err := os.Open(binPath)
	if err != nil {
		return err
	}

	bin2, err := os.OpenFile(filepath.Join(rootDir, "node"), os.O_CREATE|os.O_WRONLY, 0750)
	if err != nil {
		return err
	}

	_, err = io.Copy(bin2, bin1)
	if err != nil {
		return err
	}

	bin2, err = os.OpenFile(filepath.Join(rootDir, "npm"), os.O_CREATE|os.O_WRONLY, 0750)
	_, err = io.Copy(bin2, bin1)
	if err != nil {
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
