# node-proxy
A minimal node version manager that works as a proxy

This works by replacing nodejs binary.

## Install

```sh
go install github.com/debdutdeb/node-proxy@latest
node-proxy install
```

Once it is finished, you may need to update your bashrc or zshrc to include `export PATH="$HOME/.node-proxy/bin:$PATH"`.

Restart your shell and use node and other tools as usual. See versions automatically getting picked up, installed and ran.
