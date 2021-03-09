.PHONY: all build

WORKSPACE ?= $$(pwd)

GO_PKG_LIST := $(shell go list ./... | grep -v /vendor/ | grep -v /mock | grep -v ./pkg/apic/apiserver/clients \
	| grep -v ./pkg/apic/apiserver/models | grep -v ./pkg/apic/unifiedcatalog/models)

export GOFLAGS := -mod=vendor

all : clean

clean:
	@echo "Clean complete"

dep-check:
	@go mod verify

resolve-dependencies:
	@echo "Resolving go package dependencies"
	@go mod tidy
	@go mod vendor
	@echo "Package dependencies completed"

dep: resolve-dependencies

test:
	@go vet ${GO_PKG_LIST}
	@go test -short -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_PKG_LIST}

test-sonar:
	@go vet ${GO_PKG_LIST}
	@go test -short -coverpkg=./... -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_PKG_LIST} -json > ${WORKSPACE}/goreport.json

error-check:
	./build/scripts/error_check.sh ./pkg

sonar: test-sonar
	@echo "mode: $(mode)"
	@echo "sonarhost: $(sonarHost)"
	./sonar.sh $(mode) $(sonarHost)

lint: ## Lint the files
	@golint -set_exit_status ${GO_PKG_LIST}

apiserver-generate: # generate api server resources. ex: make apiserver-generate https apicentral.axway.com 443
	./scripts/apiserver/apiserver_generate.sh $(protocol) $(host) $(port)

unifiedcatalog-generate: ## generate unified catalog resources
	./scripts/unifiedcatalog/unifiedcatalog_generate.sh
