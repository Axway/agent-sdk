#!/bin/bash

PROTOCOL=${1:?"Protocol must be http or https"}
HOST=${2:?"Host must be set to fetch the API Server documentation, such as apicentral.axway.com"}
PORT=${3:?"Port to connect to the host"}

# set the environment vars
export GO_POST_PROCESS_FILE="`command -v gofmt` -w"
export GO111MODULE=on

node ./scripts/apiserver/generate.js $PROTOCOL $HOST $PORT

# update all go imports
goimports -w=true ./pkg/apic/apiserver

# run script to modify any files that need tweeking
./scripts/apiserver/modify_models.sh
