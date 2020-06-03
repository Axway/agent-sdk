#!/bin/bash
# pre-reqs to run this script
# 1. pip install yq
# 2 GO111MODULE=yes go get github.com/hairyhenderson/gomplate/cmd/gomplate
cat pkg/apic/apiserver/definitions/*.yaml | yq . | jq  --slurp . |
    jq -f scripts/flatten.jq |
    jq -f scripts/group.jq |
    jq -c -r -f scripts/spec_commands.jq > /tmp/run.sh
chmod +x /tmp/run.sh
bash -c /tmp/run.sh

export PROJ="git.ecd.axway.int/apigov/apic_agents_sdk"

cat pkg/apic/apiserver/definitions/*.yaml | yq . | jq  --slurp . |
    jq -f  scripts/flatten.jq | jq -c -f scripts/type_preprocess.jq |
    while read line; do
        echo "$line" |
            gomplate --context res="stdin:?type=application/json" -f scripts/resources.tmpl --out "pkg/apic/apiserver/models/$(echo $line | jq -r .group)/$(echo $line | jq -r .version)/$(echo $line | jq -r .kind).go"
    done

go fmt ./pkg/apic/apiserver/models/...
