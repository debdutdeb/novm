# NOt a node Version Manager

You could either call `novm` **not** a version manager, or, a very opinionated one. Read [PS](#ps) for my thoughts.

`novm` does not ask for a version, it tries to detect one from known sources, installs, runs them. You don't manage a new binary, you just replace `node` with this.

Only support linux and macos, see [windows support](#windows-support).

## Installation

Download as a node binary

```sh
curl -L https://github.com/debdutdeb/novm/releases/latest/download/novm-$(uname -s | tr [[:upper:]] [[:lower:]])-$(uname -m) -o /usr/local/bin/node
```

Or, as an npm binary

```sh
curl -L https://github.com/debdutdeb/novm/releases/latest/download/novm-$(uname -s | tr [[:upper:]] [[:lower:]])-$(uname -m) -o /usr/local/bin/npm
```

Either way, once run the first time, it will link itself to the other binary automatically.

Make sure you add `$HOME/.novm/bin` to your `PATH`. More [here](#install-directories).

## Updates

As I said, opinionated. Updates are automatic. For example, check the output below

```
*[main][~/Documents/Repos/node-proxy]$ node
2024/05/06 00:59:07 no nodejs version detected from sources, using latest installed
Welcome to Node.js v21.7.3.
Type ".help" for more information.
> process.stdout.write("hello world")
hello worldtrue
>
2024/05/06 00:59:19 Updating novm to v1.3.0
```

I was on v1.2.x, and got automatically upgraded to current latest. Why? Because I don't want to be bothered to update yet another tool.

## Sources

Currently the following sources are supported -
1. `NODE_VERSION` environment variable.
2. `NP_NODE_VERSION` environment variable.
3. `engines.node` under `package.json`
4. `volta.node` under `package.json`
5. `.nvmrc` file

**Contributions to more sources will be very much appreciated.**

If there is no source, `novm` will either run latest version found locally or install the latest version.

## Running

*Just run as node or npm*. That's all.

```
[/tmp/example]$ jq .engines package.json
{
  "node": "~16"
}
[/tmp/example]$ node --version
[Node v16.20.2] [==========================================================================================================================================================================================] 100.00%v16.20.2
2024/05/06 01:04:05 no new novm updates found.
```

## Install directories

`novm` installs the actual versions in `$HOME/.novm/versions` folder.

All global installs go under `$HOME/.novm/bin` folder.

## Updating actual nodejs versions

It all depends on what you have in your source. If it is an exact version, obviously won't auto update. If you have a constraint, at some point, if a new patch matches the constraint, or a new minor matches, it will be installed automatically.

## Checking out actual novm binary

Use `NOVM_WAKE=1` with `node` or `npm` call, to get `novm` options.

```
[/tmp/example]$ NOVM_WAKE=1 node --help
Usage:
  novm [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version

Flags:
  -h, --help   help for novm

Use "novm [command] --help" for more information about a command.
```

Check novm version

```
[/tmp/example]$ NOVM_WAKE=1 node version
Version: v1.3.0
GitCommit: cf13d8741fee6959d72dd8bae05cdb5750ca30e9
BuildTime: Mon May  6 00:47:46 IST 2024
```

Without `NOVM_WAKE`, you'd be calling node itself.
```
[/tmp/example]$ node version
node:internal/modules/cjs/loader:1031
  throw err;
  ^

Error: Cannot find module '/private/tmp/example/version'
    at Function.Module._resolveFilename (node:internal/modules/cjs/loader:1028:15)
    at Function.Module._load (node:internal/modules/cjs/loader:873:27)
    at Function.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:81:12)
    at node:internal/main/run_main_module:22:47 {
  code: 'MODULE_NOT_FOUND',
  requireStack: []
}
2024/05/06 01:08:26 exit status 1
```

## PS

I wrote it precisely to not have to think about node versions. There is `nvm`, `n`, `asdf`, `volta` some of which (volta) I never touched. They may and do some of them, bring more than just managing node versions, but at the end of the day I get tooling-exhaustion. Specifically since I use so many node versions across different projects, whether work, personal playing, freelance projects, doesn't matter. 

For example, at Rocket.Chat, the main repo uses node 14, another internal tool used to use node 12. So many times I forgot to make the switch. I have yet another of my own script that uses node 16. Sometimes versions are not a problem, sometimes they are, and I am tired to tracking them all the time.

There are solutions like oh-my-zsh's nvm plugin, but all of them (afaik, within my limited knowledge) are constrained to a single source, their own source ("use me" much?). There is even talk of I believe `.voltarc`. I don't want it. Not right now at least. Not unless I must. Whatever project decides to use whatever version, I just want to use node as a simple binary without always thinking about switching, no matter what project I am in, or what folder.

So this is opinionated. And I want to keep it that way. There isn't much to configure, nor am I planning to add.

Please feel free to contribute. What brings more joy than that? Not many things in this world.

## Windows support

I don't have a windows machine with me. But will accept PRs.
