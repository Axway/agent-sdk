#!/bin/bash

# openapi-generator generate -g go -i ./pkg/apic/apiserver/apiserver-spec.json --package-name v1 --output ./pkg/apic/apiserver -DmodelDocs=false -Dmodels
# node ./scripts/generate_long.js
# # goimports -w=true ./pkg/apic/apiserver/models
# go fmt ./pkg/apic/apiserver/models/...

yq . pkg/apic/apiserver/definitions/*.yaml | jq  --slurp . |
    jq -f  scripts/flatten.jq | jq -c -f scripts/type_preprocess.jq |
    while read line; do
      echo "$line"
      # gomplate --context res="stdin:?type=application/json" -f scripts/resources.tmpl --out "pkg/apic/apiserver/models/$(echo $line | jq -r .group)/$(echo $line | jq -r .version)/$(echo $line | jq -r .kind).go"
    done || { echo "$0: failed to run gomplate spec"; exit 1; }

# go fmt ./pkg/apic/apiserver/models/...