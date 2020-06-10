#!/bin/bash

node ./scripts/apiserver/generate.js
go fmt ./pkg/apic/apiserver/...
goimports -w=true ./pkg/apic/apiserver/models
