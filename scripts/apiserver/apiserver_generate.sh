#!/bin/bash

# set thes environment vars
export GO_POST_PROCESS_FILE="`command -v gofmt` -w"
export GO111MODULE=on

node ./scripts/apiserver/generate.js

# just in case, update all go imports
goimports -w=true ./pkg/apic/apiserver/models

# run script to modify any files that need tweeking
./scripts/apiserver/modify_models.sh
