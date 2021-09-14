.PHONY: all build

WORKSPACE ?= $$(pwd)

GO_TEST_LIST := $(shell go list ./... | grep -v /mock)

GO_PKG_LIST := $(shell go list ./... | grep -v /mock | grep -v ./pkg/apic/apiserver/clients \
	| grep -v ./pkg/apic/apiserver/models | grep -v ./pkg/apic/unifiedcatalog/models)

export GOFLAGS := -mod=mod

all : clean

clean:
	@echo "Clean complete"

dep-check:
	@go mod verify

resolve-dependencies:
	@echo "Resolving go package dependencies"
	@go mod tidy
	@echo "Package dependencies completed"

dep: resolve-dependencies

test: dep
	@go vet ${GO_TEST_LIST}
	@go test -race -short -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_TEST_LIST}

test-sonar: dep
	@go vet ${GO_PKG_LIST}
	@go test -short -coverpkg=./... -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_PKG_LIST} -json > ${WORKSPACE}/goreport.json

error-check:
	./build/scripts/error_check.sh ./pkg

sonar: test-sonar
	./build/scripts/sonar.sh $(mode) $(sonarHost)

lint: ## Lint the files
	@golint -set_exit_status ${GO_PKG_LIST}

apiserver-generate: # generate api server resources, prod by default. ex: make apiserver-generate protocol=https host=apicentral.axway.com port=443
	docker run --rm -v $(shell pwd)/scripts/apiserver:/codegen/scripts -v $(shell pwd)/pkg/apic/apiserver:/codegen/output -e PROTOCOL='$(protocol)' -e HOST='$(host)'  -e PORT='$(port)' -e USERID=$(shell id -u) -e GROUPID=$(shell id -g) -w /codegen/scripts --entrypoint ./apiserver_generate.sh ampc-beano-docker-snapshot-phx.artifactory-phx.ecd.axway.int/beano-alpine-codegen:latest

unifiedcatalog-generate: ## generate unified catalog resources
	./scripts/unifiedcatalog/unifiedcatalog_generate.sh
