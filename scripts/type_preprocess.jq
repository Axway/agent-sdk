# this jq filter takes an array of flattened resources and
# creates the context input for the gomplate

# checks if a resource spec is empty
def isEmptySpec: . != {"type":"object", "additionalProperties": false};

.[] | {kind, version, group, scope, "resource": .names.plural, "fields" : ({"spec": .schema.openAPIV3Schema | isEmptySpec } + (. | .subresources // {} | with_entries({key, "value": .value.openAPIV3Schema | isEmptySpec })))}
