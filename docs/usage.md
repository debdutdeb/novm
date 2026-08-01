# Using novm

`novm` is not a version manager in the traditional sense — there is no `novm use <version>` command you need to remember to run. Instead, the `node`/`npm`/`npx`/`yarn`/`pnpm`/`corepack` binaries on your `PATH` *are* `novm`. Every time one of them runs, `novm` detects which Node.js version the current project wants, makes sure it's installed, and hands off to it. You never think about switching versions again.

This document covers day-to-day usage. If you want to use novm's Node-management logic as a Go library instead of a CLI tool, see [Using novm as a Go module](go-module.md).

## Installation

Download the binary and place it on your `PATH` under the name of the tool you want it to replace — `node` or `npm` (it will link the rest of the tool names itself the first time it runs):

```sh
curl -L https://github.com/debdutdeb/novm/releases/latest/download/novm-$(uname -s | tr '[[:upper:]]' '[[:lower:]]')-$(uname -m) -o ~/.local/bin/node && chmod +x ~/.local/bin/node
```

or, for `npm`:

```sh
curl -L https://github.com/debdutdeb/novm/releases/latest/download/novm-$(uname -s | tr '[[:upper:]]' '[[:lower:]]')-$(uname -m) -o ~/.local/bin/npm && chmod +x ~/.local/bin/npm
```

On first run, novm symlinks itself to `node`, `npm`, `npx`, `yarn`, `corepack`, and `pnpm` in the same directory. Install it somewhere your user already owns (e.g. `~/.local/bin`) so this linking step doesn't need `sudo`. If linking fails on that first run, novm will not retry automatically — see [Manual linking](#manual-linking) below.

Only Linux and macOS are supported. See [Windows support](../README.md#windows-support).

Make sure `$HOME/.novm/bin` is on your `PATH` too — that's where globally installed packages (like `yarn`/`pnpm`, which novm installs on demand) live. See [Install directories](#install-directories).

### Manual linking

If the automatic symlinking didn't happen (e.g. installed as root, or into a directory you don't own), link the binaries yourself before first use:

```sh
sudo ln -s $(which node) $(dirname $(which node))/npm
# or, if you downloaded it as npm instead
sudo ln -s $(which npm) $(dirname $(which npm))/node
```

## How version detection works

Every time `node`/`npm`/etc. runs, novm walks up from the current directory (2 levels by default, see [`NOVM_DEPTH_SOURCE_DETECTION`](#environment-variables)) looking for a Node version in the following sources, in order:

| Priority | Source | Notes |
|---|---|---|
| 1 | `NODE_VERSION` environment variable | |
| 2 | `NP_NODE_VERSION` environment variable | Deprecated, prefer `NODE_VERSION` |
| 3 | `engines.node` in `package.json` | |
| 4 | `volta.node` in `package.json` | |
| 5 | `.nvmrc` | |
| 6 | `.node-version` | Same format as `.nvmrc`; also read by nodenv, fnm, Volta, asdf |
| 7 | `.tool-versions` | asdf/mise format, reads the `nodejs` line. **Experimental.** |
| 8 | `Dockerfile` | Reads the Node version out of a `FROM node:<version>` line. **Experimental.** |

The version value can be an exact version (`16.20.2`) or a semver range/constraint (e.g. `~16`, `>=18 <21`) — novm resolves it against the current Node.js release index.

Experimental sources log a warning when they match, since their detection is less battle-tested than the others.

If none of the sources produce a version, novm falls back to the latest version you already have installed, or downloads the latest release if nothing is installed yet:

```
$ node
2024/05/06 00:59:07 no nodejs version detected from sources, using latest installed
Welcome to Node.js v21.7.3.
```

## Running

Just run `node`, `npm`, `npx`, `yarn`, `pnpm`, or `corepack` as you normally would. If the resolved version isn't installed yet, novm downloads it first (with a progress bar when running interactively):

```
$ jq .engines package.json
{
  "node": "~16"
}
$ node --version
[Node v16.20.2] [========================================] 100.00%
v16.20.2
```

`yarn` and `pnpm` aren't bundled with Node.js releases, so the first time either is invoked for a given Node install, novm installs it globally via `npm install -g` before running it.

## Updates

novm updates itself automatically — there's no `novm upgrade` command. On every invocation it checks (at most once a minute, backing off further over time) whether a newer release is available on GitHub, downloads it in the background while your command runs, and swaps the binary in afterwards:

```
$ node
2024/05/06 00:59:19 Updating novm to v1.3.0
```

Signals like `SIGINT`/`SIGTERM` are briefly deferred while the binary swap is in progress so an update can't leave you with a corrupt install.

Node.js versions themselves only update when your source resolves to a new version — an exact pin (`16.20.2`) never moves, but a range (`~16`, `lts`) will pick up new matching releases as they're published.

## Install directories

| Path | Contents |
|---|---|
| `$HOME/.novm/versions` | Installed Node.js versions, one directory per version |
| `$HOME/.novm/bin` | Global installs (e.g. `yarn`, `pnpm`) |
| `$HOME/.novm/state.json` | novm's own state: update-check timestamps, per-version usage stats |
| `$HOME/.novm/node_versions.json` | Cached copy of the Node.js release index (refreshed daily) |

Override the root (`$HOME/.novm`) with the `NOVM_WORKDIR` environment variable.

### Automatic cache cleanup

novm periodically (at most once every 24 hours) looks at installed versions and removes ones that have gone unused for 10+ days *and* weren't averaging more than 10 uses per 3 days while they were active. This keeps `~/.novm/versions` from growing unbounded if you bounce between many project versions, without evicting versions you use often.

## The `novm` CLI

> This is still currently experimental. `novm` **may** switch to a more non-traditional wake-only model of cli in the future. Proposed plan is something like `NOVM_WAKE=cli node help` to "wake" the novm cli instead of the current `1`. Should also be able to pass `NOVM_WAKE=cli,state node` to accumulate wake codes, the most immediate takes effect. And `NOVM_WAKE{int}` is the final form, where wake commands can be composed, e.g. `NOVM_WAKE1=cli NOVM_WAKE2=state node` to wake both the cli and state dump. This is a bit more flexible than the current `NOVM_WAKE=1` vs `NOVM_WAKE=state` dichotomy. You can also set default wake commands that way which don't wake `novm` cli directly, `NOVM_WAKE999='add_command,gemini-cli'` in a project's `.envrc` to automatically install gemini cli or make it available in the project without adding it globally elsewhere, and still able to pass more wake commands as usual.

Normally, `node`/`npm`/etc. go straight to running Node.js — they never show you a `novm` command interface. To reach the underlying `novm` CLI instead, set `NOVM_WAKE=1`:

```
$ NOVM_WAKE=1 node --help
Usage:
  novm [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  setup       Re-run first-install setup (npm prefix + binary symlinks)
  version     Print the novm version, commit, and build time
  where       Get the on-disk location of an installed version

Flags:
  -h, --help   help for novm

Use "novm [command] --help" for more information about a command.
```

Without `NOVM_WAKE`, `node version` (for example) is interpreted as "run the file `version` with Node.js", not as the novm CLI:

```
$ node version
node:internal/modules/cjs/loader:1031
  throw err;
Error: Cannot find module '/private/tmp/example/version'
```

### `novm version`

```
$ NOVM_WAKE=1 node version
Version: v1.3.0
GitCommit: cf13d8741fee6959d72dd8bae05cdb5750ca30e9
BuildTime: Mon May  6 00:47:46 IST 2024
```

### `novm where <version>`

Prints the on-disk path of an installed version (aliases: `which`, `locate`, `find`):

```
$ NOVM_WAKE=1 node where 16.20.2
/home/you/.novm/versions/16.20.2/linux/x64
```

### `novm setup`

Re-runs the first-install steps (setting the `npm` prefix in `~/.npmrc` and symlinking `node`/`npm`/`npx`/`yarn`/`corepack`/`pnpm`). Useful if the automatic linking on first run didn't complete, without needing to delete your state file.

### Debugging: dumping internal state

`NOVM_WAKE=state` (instead of `1`) prints novm's internal `~/.novm/state.json` as formatted JSON — update-check timestamps and per-version usage stats used for automatic cache cleanup — instead of running the CLI or Node.js:

```
$ NOVM_WAKE=state node
{
  "update": { "lastChecked": "...", "timesChecked": 3 },
  "poolControl": { "usage": { "16.20.2": { "hits": 12, "lastUsed": "..." } } }
}
```

This is a debugging aid, not a stable interface — don't script against its shape.

## Environment variables

| Variable | Effect |
|---|---|
| `NODE_VERSION` | Highest-priority version source. |
| `NP_NODE_VERSION` | Deprecated alias for `NODE_VERSION`. |
| `NOVM_WAKE` | Set to `1` to talk to the `novm` CLI instead of Node.js/npm. |
| `NOVM_WORKDIR` | Overrides novm's root directory (default `$HOME/.novm`). |
| `NOVM_DEPTH_SOURCE_DETECTION` | How many parent directories to search for a version source (default `2`). |

## Troubleshooting

- **"no nodejs version detected from sources, using latest installed"** — none of the sources in the table above matched anywhere up the directory tree; this is informational, not an error.
- **A stale symlink after install** — re-run `NOVM_WAKE=1 node setup`.
- **Wrong version keeps getting picked** — remember sources are checked in priority order (env vars beat `package.json` beat `.nvmrc`, etc.) and novm searches parent directories too; check `NOVM_DEPTH_SOURCE_DETECTION` if a source further up the tree is winning unexpectedly.
