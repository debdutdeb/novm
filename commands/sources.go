package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type source map[string]func(string) (string, error)

func sourceEnvironment(_dir string) (string, error) {
	if version := os.Getenv("NODE_VERSION"); version != "" {
		return version, nil
	}

	version := os.Getenv("NP_NODE_VERSION")
	if version != "" {
		log.Println("NP_NODE_VERSION is deprecated, use NODE_VERSION instead")
	}

	return version, nil
}

type packageJson struct {
	Engines struct {
		Node string `json:"node"`
	} `json:"engines"`

	Volta struct {
		Node string `json:"node"`
	} `json:"volta"`
}

func sourcePackageJson(dir string) (string, error) {
	var p packageJson
	f, err := os.Open(filepath.Join(dir, "package.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to open package.json: %w", err)
	}

	defer f.Close()

	err = json.NewDecoder(f).Decode(&p)
	if err != nil {
		return "", fmt.Errorf("failed to read package.json: %w", err)
	}

	if p.Engines.Node != "" {
		return p.Engines.Node, nil
	}

	if p.Volta.Node != "" {
		return p.Volta.Node, nil
	}

	return "", nil
}

func sourceNvmrc(dir string) (string, error) {
	// only supports the version
	f, err := os.Open(".nvmrc")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to read .nvmrc: %w", err)
	}

	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read .nvmrc: %w", err)
	}

	if b[len(b)-1] == 10 {
		b = b[:len(b)-1]
	}

	return string(b), nil
}
