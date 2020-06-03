.[] | "openapi-generator generate -g go -i /dev/stdin --package-name \(.version) --output pkg/apic/apiserver/models/\(.group)/\(.version) -DmodelDocs=false -Dmodels << 'EOF'\n\(.openapi)\nEOF"
