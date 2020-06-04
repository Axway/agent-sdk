#!/bin/bash

openapi-generator generate -g go -i ./pkg/apic/apiserver/apiserver-spec.json --package-name v1 --output ./pkg/apic/apiserver -DmodelDocs=false -Dmodels
node ./scripts/generate.js
goimports -w=true ./pkg/apic/apiserver/models
go fmt ./pkg/apic/apiserver/models/...