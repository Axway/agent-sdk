.PHONY: all build

WORKSPACE ?= $$(pwd)
RACEFLAG ?= -race

GO_TEST_LIST := $(shell go list ./... | grep -v /mock)

GO_PKG_LIST := $(shell go list ./... | grep -v /mock | grep -v ./pkg/apic/apiserver/clients \
	| grep -v ./pkg/apic/apiserver/models)

export GOFLAGS := -mod=mod

PROTO_OUT_PATH := ${WORKSPACE}

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
	@go test -v ${RACEFLAG} -short -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_TEST_LIST} | tee test-output.log
	@go-junit-report -in test-output.log -iocopy -set-exit-code -out report.xml

test-sonar: dep
	@go vet ${GO_PKG_LIST}
	@go test -short -coverpkg=./... -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_PKG_LIST} -json > ${WORKSPACE}/goreport.json

apiserver-generate: # generate api server resources, prod by default. ex: make apiserver-generate protocol=https host=apicentral.axway.com port=443
	docker run --name generator --rm -v $(shell pwd)/scripts/apiserver:/codegen/scripts -v $(shell pwd)/pkg/apic/apiserver:/codegen/output -e PROTOCOL='$(protocol)' -e HOST='$(host)'  -e PORT='$(port)' -e DEBUG='$(debug)' -e USERID=$(shell id -u) -e GROUPID=$(shell id -g) -w /codegen/scripts --entrypoint ./apiserver_generate.sh ampc-beano-docker-release-phx.artifactory-phx.ecd.axway.int/base-images/beano-alpine-codegen:latest

PROTOFILES := $(shell find $(WORKSPACE)/proto -type f -name '*.proto')
PROTOTARGETS := $(PROTOFILES:.proto=.pb.go)

%.pb.go : %.proto
	@echo $<
	@docker run --rm -u $(shell id -u) \
	  -v${WORKSPACE}:${WORKSPACE} \
	  -w${WORKSPACE} rvolosatovs/protoc:latest \
	  --proto_path=${WORKSPACE}/proto --go_out=${PROTO_OUT_PATH} --go-grpc_out=${PROTO_OUT_PATH} \
	  $<

# generate protobufs
protoc: $(PROTOTARGETS)
