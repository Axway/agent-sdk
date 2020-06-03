# this jq filter takes the group/version grouping and generates
# openap-generator commands to create the models corresponding to each group

.[] | "openapi-generator generate -g go -i /dev/stdin --package-name \(.version) --output pkg/apic/apiserver/models/\(.group)/\(.version) -DmodelDocs=false -Dmodels << 'EOF'\n\(.openapi)\nEOF"
