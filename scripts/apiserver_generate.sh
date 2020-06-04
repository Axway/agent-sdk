#!/bin/bash

node ./scripts/generate.js
go fmt ./pkg/apic/apiserver/models/...
goimports -w=true ./pkg/apic/apiserver/models