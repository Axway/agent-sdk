#!/bin/bash

# set thes environment vars
export GO_POST_PROCESS_FILE="`command -v gofmt` -w"
export GO111MODULE=on

node ./scripts/unifiedcatalog/generate.js

## just in case, update all go imports
goimports -w=true ./pkg/apic/unifiedcatalog/models
