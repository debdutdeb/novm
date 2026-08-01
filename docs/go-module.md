# Using novm as a Go module

Most of what makes novm work — resolving a version string, downloading a Node.js release, and running `node`/`npm`/`npx`/`yarn`/`pnpm`/`corepack` against it — lives in a regular importable Go package: `github.com/debdutdeb/novm/v3/pkg/n`. You can use it to drive Node.js installs and invocations from your own Go programs, without going through the `novm` CLI at all.

This document covers the library API. For the CLI/end-user tool itself, see [Using novm](usage.md).

## Installing

```sh
go get github.com/debdutdeb/novm/v3
```

The module requires Go 1.25+ (see `go.mod`).

```go
import "github.com/debdutdeb/novm/v3/pkg/n"
```

> There is also a `github.com/debdutdeb/novm/v3/pkg` package (no `/n`). It has the same API but every exported symbol is marked `Deprecated` in favor of `pkg/n`. Don't use it for new code.

## Core type: `n.N`

`n.N` represents a single resolved Node.js installation. Construct one with `NewNodeManager`:

```go
func NewNodeManager(global bool, version string, rootDir string) (*N, error)
```

- `global` — if `true`, novm assumes Node.js is already installed and on `PATH` (via `exec.LookPath("node")`) and wraps that installation, rather than managing its own copy under `rootDir`. If `true` and no `node` is found on `PATH`, it returns `n.ErrNodeNotInstalled`.
- `version` — one of:
  - `"latest"` — resolves to the newest available Node.js release for your OS/arch.
  - `"lts"` — resolves to the newest LTS release.
  - an exact version, e.g. `"18.20.4"`.
  - a semver constraint, e.g. `"~18"`, `">=16 <21"` — resolved against the Node.js release index, picking the newest matching release that ships a build for your platform.
- `rootDir` — where novm stores downloaded versions (`<rootDir>/versions/<version>/<GOOS>/<GOARCH>`) and its release-index cache (`<rootDir>/node_versions.json`, refreshed once every 24 hours). This is the same directory the CLI calls `$HOME/.novm`, but you can point it anywhere.

```go
manager, err := n.NewNodeManager(false, "~18", "/tmp/my-node-cache")
if err != nil {
    log.Fatal(err)
}
```

## Installing and running Node.js

```go
if err := manager.EnsureInstalled(); err != nil {
    log.Fatal(err)
}

if err := manager.Run("script.js", "--flag"); err != nil {
    log.Fatal(err)
}
```

- `EnsureInstalled() error` — installs the resolved version if it isn't already present under `rootDir`. Safe to call every time; it's a no-op if already installed.
- `Install() error` — downloads and installs the resolved version unconditionally (used internally by `EnsureInstalled`; call directly only if you want to force a re-install).
- `Run(args ...string) error` — execs `node` with the given args, connecting stdin/stdout/stderr to the current process (like a shell would). Blocks until the child exits.
- `CaptureOutput(args ...string) (stdout, stderr []byte, err error)` — runs `node` and captures output instead of streaming it. `Deprecated` in favor of `Experimental_UnderlyingStdCmd` for new code that needs more control.
- `Version() string` — runs `node --version` against the resolved binary and returns the trimmed output.

## npm, yarn, pnpm, npx, corepack

`N` exposes companion runners for the rest of the Node.js toolchain, each sharing the same resolved version/environment:

```go
if err := manager.Npm().Run("install"); err != nil {
    log.Fatal(err)
}

if err := manager.Npx().Run("some-cli", "--version"); err != nil {
    log.Fatal(err)
}
```

- `Npm() Npm`, `Npx() Npx`, `Corepack() Corepack` — these ship alongside Node.js itself, so no separate install step is needed.
- `Yarn() Yarn`, `Pnpm() Pnpm` — these are **not** bundled with Node.js. Their binaries are expected at `<rootDir>/bin/yarn` and `<rootDir>/bin/pnpm`; installing them there (e.g. via `manager.Npm().Run("install", "yarn", "-g")`) is your responsibility when using the library directly (The CLI does this automatically on first use).

All five (`Npm`, `Yarn`, `Npx`, `Corepack`, `Pnpm`) implement the same small interface:

```go
Run(args ...string) error
CaptureOutput(args ...string) ([]byte, []byte, error)
Experimental_UnderlyingStdCmd(args ...string) *exec.Cmd
```

## Escape hatch: `Experimental_UnderlyingStdCmd`

When `Run`/`CaptureOutput` aren't flexible enough (e.g. you need to set a working directory, pipe output somewhere custom, or run asynchronously), get the underlying `*exec.Cmd` directly:

```go
cmd := manager.Npm().Experimental_UnderlyingStdCmd("install", "some-package")
cmd.Dir = "/path/to/project"
if err := cmd.Start(); err != nil {
    log.Fatal(err)
}
if err := cmd.Wait(); err != nil {
    log.Fatal(err)
}
```

The returned `*exec.Cmd` already has the correct binary path and environment (including `PATH` pointed at the resolved Node install and `NODE_VERSION` set) — it just isn't wired to stdio or started yet. As the name signals, this method's exact shape isn't guaranteed to stay stable across releases, but it adds no state to `N` itself, so it's safe to use without side effects.

## Apple Silicon note

Node.js versions below 16 don't ship `arm64` macOS binaries. `NewNodeManager` already accounts for this automatically when resolving `"latest"`/`"lts"`/constraints on `darwin`. If you construct a manager for an exact old version on an M-series Mac and need to force the Intel build (run under Rosetta), call:

```go
manager.SetBinaryArchX86()
```

before `Install()`/`EnsureInstalled()`.

## Supporting packages

These aren't required to use `pkg/n`, but are part of the same module and may be useful if you're embedding more of novm's behavior:

- `github.com/debdutdeb/novm/v3/versions` — three build-time-injected string variables: `Version`, `GitCommit`, `BuildTime`. Only meaningful in binaries built via the project's `Makefile` (`go build -ldflags ...`); empty otherwise.
- `github.com/debdutdeb/novm/v3/common` — path helpers built around a `RootDir` global (`Where(version)`, `VersionsDir()`, `ListVersions()`, `InRootDir(dir)`). These assume the CLI's directory layout and read `NOVM_WORKDIR`/`HOME` at `init()` time, so they're more coupled to the CLI than to `pkg/n` — most library users won't need them.

## Full example

```go
package main

import (
    "log"

    "github.com/debdutdeb/novm/v3/pkg/n"
)

func main() {
    manager, err := n.NewNodeManager(false, "20", "/tmp/novm-example")
    if err != nil {
        log.Fatal(err)
    }

    if err := manager.EnsureInstalled(); err != nil {
        log.Fatal(err)
    }

    log.Println("using Node.js", manager.Version())

    if err := manager.Run("-e", "console.log('hello from a managed Node.js install')"); err != nil {
        log.Fatal(err)
    }
}
```
