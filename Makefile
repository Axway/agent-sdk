.PHONY: all build

WORKSPACE ?= $$(pwd)

GO_PKG_LIST := $(shell go list ./... | grep -v /vendor/ | grep -v /mock | grep -v ./pkg/apic/apiserver/**/definitions/v1alpha  | grep -v ./pkg/apic/apiserver/**/management/v1alpha)
all : clean

clean:
	@echo "Clean complete"

dep-update:
  @go mod tidy

resolve-dependencies:
	@echo "Resolving go package dependencies"
	@go mod download
	@echo "Package dependencies completed"

dep: resolve-dependencies

test: dep
	@go vet ${GO_PKG_LIST}
	@go test -short -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_PKG_LIST}

test-sonar: dep
	@go vet ${GO_PKG_LIST}
	@go test -short -coverpkg=./... -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_PKG_LIST} -json > ${WORKSPACE}/goreport.json

sonar: test-sonar
	sonar-scanner -X \
		-Dsonar.host.url=http://quality1.ecd.axway.int \
		-Dsonar.language=go \
		-Dsonar.projectName=APIC_AGENTS_SDK \
		-Dsonar.projectVersion=1.0 \
		-Dsonar.projectKey=APIC_AGENTS_SDK \
		-Dsonar.sourceEncoding=UTF-8 \
		-Dsonar.projectBaseDir=${WORKSPACE} \
		-Dsonar.sources=. \
		-Dsonar.tests=. \
		-Dsonar.exclusions=**/mock/**,**/vendor/**,**/definitions/v1alpha1/**,**/management/v1alpha1/** \
		-Dsonar.test.inclusions=**/*test*.go \
		-Dsonar.go.tests.reportPaths=goreport.json \
		-Dsonar.go.coverage.reportPaths=gocoverage.out

lint: ## Lint the files
	@golint -set_exit_status $(shell go list ./... | grep -v /vendor/ | grep -v /mock | grep -v ./pkg/apic/apiserver/models/management | grep -v ./pkg/apic/apiserver/models/definitions)

apiserver_generate: ## generate api server resources
	./scripts/apiserver/apiserver_generate.sh
