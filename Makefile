build:
	CGO_ENABLED=0 go build -ldflags="-X 'github.com/debdutdeb/node-proxy/versions.BuildTime=$(shell date -u)' -X 'github.com/debdutdeb/node-proxy/versions.GitCommit=$(shell git rev-parse HEAD)' -extldflags '-static'" -o node .

install: build
	sudo cp node ~/.local/bin

.PHONY: build install
