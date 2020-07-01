#!/bin/bash

node ./scripts/unifiedcatalog/generate.js
go fmt ./pkg/apic/unifiedcatalog/models/...
goimports -w=true ./pkg/apic/unifiedcatalog/models
