package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type source map[string]func() (string, error)

func sourceEnvironment() (string, error) {
	return os.Getenv("NP_NODE_VERSION"), nil
}

type packageJson struct {
	Engines struct {
		Node string `json:"node"`
	} `json:"engines"`

	Volta struct {
		Node string `json:"node"`
	} `json:"volta"`
}

func sourcePackageJson() (string, error) {
	var p packageJson
	f, err := os.Open("package.json")
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
		return p.Engines.Node, nil
	}

	return "", nil
}

func sourceNvmrc() (string, error) {
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

