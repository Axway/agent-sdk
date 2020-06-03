#!/bin/bash
# pre-reqs to run this script
# 1. pip install yq
# 2. GO111MODULE=on go get github.com/hairyhenderson/gomplate/cmd/gomplate (outside a go project)

# this script takes a set of api-server resource definition files
# and generates go structs for each resource version

# the resource files are processed with jq filters
# and grouped by group/version
# all resource spec and subresources in a group/version result in an openapi spec that's passed to openapi-generator
# openapi-generator will generate go structs for each of them

# each group/version ends up in a pkg
# pkg/apiserver/models/<group>/<version>

set -o pipefail

yq . pkg/apic/apiserver/definitions/*.yaml | jq  --slurp . |
    jq -f scripts/flatten.jq |
    jq -f scripts/group.jq |
    jq -c -r -f scripts/spec_commands.jq > /tmp/run.sh || { echo "$0: failed to generate openapi-generator comands"; exit 1; }

chmod +x /tmp/run.sh
bash -c /tmp/run.sh

# additionally, the main struct for each resources
# is generated via gomplate

export PROJ="git.ecd.axway.int/apigov/apic_agents_sdk"

# gomplate is used to generate the resource structs that

yq . pkg/apic/apiserver/definitions/*.yaml | jq  --slurp . |
    jq -f  scripts/flatten.jq | jq -c -f scripts/type_preprocess.jq |
    while read line; do
        echo "$line" |
            gomplate --context res="stdin:?type=application/json" -f scripts/resources.tmpl --out "pkg/apic/apiserver/models/$(echo $line | jq -r .group)/$(echo $line | jq -r .version)/$(echo $line | jq -r .kind).go"
    done || { echo "$0: failed to run gomplate spec"; exit 1; }

go fmt ./pkg/apic/apiserver/models/...
