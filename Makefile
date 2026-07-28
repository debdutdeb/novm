GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

BIN ?= node

COMMIT ?= $(shell git rev-parse HEAD)

BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

VERSION ?= "develop"

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
				go build -ldflags="-X 'github.com/debdutdeb/novm/v3/versions.Version=$(VERSION)' -X 'github.com/debdutdeb/novm/v3/versions.BuildTime=$(BUILD_TIME)' -X 'github.com/debdutdeb/novm/v3/versions.GitCommit=$(COMMIT)' -extldflags '-static'" -o $(BIN) .

install: build
	@sudo cp $(BIN) ~/.local/bin

.PHONY: build install
