package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/debdutdeb/node-proxy/cmd"
	"github.com/debdutdeb/node-proxy/common"
	"github.com/debdutdeb/node-proxy/pkg"

	"golang.org/x/mod/semver"
)

var NodeJsVersion string = ""

func init() {
	if filepath.Base(os.Args[0]) == common.BIN_NAME {
		return
	}

	sources := source{
		"environment":  sourceEnvironment, // NP_NODE_VERSION
		"package.json": sourcePackageJson, // engines, volta
		"nvmrc":        sourceNvmrc,
	}

	for sourceName, sourceFn := range sources {
		v, err := sourceFn()
		if v != "" {
			NodeJsVersion = v
			return
		}

		if err != nil {
			log.Println("unable to parse source:", sourceName, "error:", err)
		}
	}
}

func Run() error {
	var err error

	root, err := common.RootDir()
	if err != nil {
		return err
	}

	if os.Getenv("NOVM_WAKE") != "" {
		return cmd.Root(root).Execute()
	}

	if NodeJsVersion == "" {
		log.Println("no nodejs version detected from sources, using latest installed")

		NodeJsVersion, err = findMaxInstalledVersion(filepath.Join(root, "versions"))
		if err != nil {
			return fmt.Errorf("failed to detect current nodejs version: %w", err)
		}
	}

	n, err := pkg.NewNodeManager(false, NodeJsVersion, root)
	if err != nil {
		return fmt.Errorf("failed to initialize node manager: %w", err)
	}

	err = n.EnsureInstalled()
	if err != nil {
		return fmt.Errorf("failed to install node version %w", err)
	}

	if filepath.Base(os.Args[0]) == "npm" {
		return n.Npm().Run(os.Args[1:]...)
	}

	return n.Run(os.Args[1:]...)
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
