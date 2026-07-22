package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/debdutdeb/novm/v3/cmd"
	"github.com/debdutdeb/novm/v3/common"
	"github.com/debdutdeb/novm/v3/pkg"

	"golang.org/x/mod/semver"
)

var NodeJsVersion string = ""

func init() {
	if filepath.Base(os.Args[0]) == common.BIN_NAME {
		return
	}

	sources := source{
		sourceEnvironmentVariable: sourceEnvironment, // NODE_VERSION
		sourcePackageJsonFile:     sourcePackageJson, // engines, volta
		sourceNvmFile:             sourceNvmrc,
		sourceNodeVersionFile:     sourceNodeVersion,
		sourceToolVersionsFile:    wrapInExperimental(sourceToolVersionsFile, sourceToolVersions), // asdf, mise
		sourceDockerfileFile:      wrapInExperimental(sourceDockerfileFile, sourceDockerfile),
	}

	depth := common.DepthSourceDetection()
	dir := "."
	for i := 0; i <= depth; i++ {
		for sourceName, sourceFn := range sources {
			v, err := sourceFn(dir)
			if v != "" {
				NodeJsVersion = v
				return
			}

			if err != nil {
				log.Println("unable to parse source:", sourceName, "error:", err)
			}
		}
		dir = filepath.Join(dir, "..")
	}
}

func Run() error {
	var err error

	root := common.RootDir

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

	switch filepath.Base(os.Args[0]) {
	case "npm":
		return n.Npm().Run(os.Args[1:]...)
	case "yarn":
		if err := installIfNotExists(n, "yarn"); err != nil {
			return err
		}
		return n.Yarn().Run(os.Args[1:]...)
	case "npx":
		return n.Npx().Run(os.Args[1:]...)
	case "corepack":
		return n.Corepack().Run(os.Args[1:]...)
	case "pnpm":
		if err := installIfNotExists(n, "pnpm"); err != nil {
			return err
		}
		return n.Pnpm().Run(os.Args[1:]...)
	}

	return n.Run(os.Args[1:]...)
}

func installIfNotExists(n *pkg.N, bin string) error {
	path := filepath.Join(common.RootDir, "bin", bin)
	fst, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// install
			return n.Npm().Run("install", bin, "-g")
		}

		return err
	}
	if fst.IsDir() {
		return fmt.Errorf("%s is a dir, binary was expected", path)
	}
	return nil
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
