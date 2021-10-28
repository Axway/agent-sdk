PROJECT_NAME := watchclient
PKG_LIST := $(shell go list ./... | grep -v /vendor/)

BIN=watchclient

.PHONY: clean

_all: clean lint build ## Build everything

all: clean build

lint: ## Lint the files
	@golint	-set_exit_status	${PKG_LIST}

build: ## Build the binary for linux
	CGO_ENABLED=0 GOARCH=amd64 go build -o ./bin/$(BIN)

clean: ## Clean out dir
	rm -rf ./bin

help: ## Display this help screen
	@grep	-h	-E	'^[a-zA-Z_-]+:.*?## .*$$'	$(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

