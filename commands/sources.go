package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type sourceType = string

const (
	sourceNvmFile             sourceType = ".nvmrc"
	sourcePackageJsonFile     sourceType = "package.json"
	sourceEnvironmentVariable sourceType = "environment"
	sourceNodeVersionFile     sourceType = ".node-version"
	sourceToolVersionsFile    sourceType = ".tool-versions"
	sourceDockerfileFile      sourceType = "Dockerfile"
)

func wrapInExperimental(source sourceType, fn func(string) (string, error)) func(string) (string, error) {
	return func(dir string) (string, error) {
		v, err := fn(dir)
		if err != nil || v == "" {
			return "", err
		}

		log.Printf("%s is an experimental source, and may not be behaving as expected, disable with NOVM_NO_EXPERIMENTAL=1", source)
		return v, nil
	}
}

func openpath(dir, name string) (path string, file io.ReadCloser, err error) {
	path = filepath.Join(dir, name)
	file, err = os.Open(path)
	return
}

type source map[sourceType]func(string) (string, error)

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
	_, f, err := openpath(dir, "package.json")
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
	_, f, err := openpath(dir, ".nvmrc")
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

	return strings.TrimSpace(string(b)), nil
}

func sourceNodeVersion(dir string) (string, error) {
	// same format as .nvmrc, supported by nodenv, n, fnm, Volta, asdf
	f, err := os.Open(filepath.Join(dir, ".node-version"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to read .node-version: %w", err)
	}

	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read .node-version: %w", err)
	}

	return strings.TrimSpace(string(b)), nil
}

func sourceToolVersions(dir string) (string, error) {
	// asdf/mise format: lines of "<plugin> <version> [<version>...]"
	f, err := os.Open(filepath.Join(dir, ".tool-versions"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to read .tool-versions: %w", err)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nodejs" {
			return fields[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read .tool-versions: %w", err)
	}

	return "", nil
}

func sourceDockerfile(dir string) (string, error) {
	path, f, err := openpath(dir, "Dockerfile")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to open Dockerfile %s %v", path, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	from := ""
	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.Split(strings.ToLower(strings.TrimSpace(text)), " ")
		if parts[0] == "from" {
			from = parts[1]
		}
	}
	parts := strings.Split(from, ":")
	if strings.HasPrefix(parts[0], "docker.io/") {
		parts[0] = strings.TrimLeft(parts[0], "docker.io/")
	}
	if parts[0] != "node" {
		return "", nil
	}
	version := strings.Split(parts[1], "-")[0]
	return version, nil
}
