# checks if a resource spec is empty
def isEmptySpec: . != {"type":"object", "additionalProperties": false};

.[] | {kind, version, group, scope, "resource": .names.plural, "fields" : ({"spec": .schema.openAPIV3Schema | isEmptySpec } + (. | .subresources // {} | with_entries({key, "value": .value.openAPIV3Schema | isEmptySpec })))}
