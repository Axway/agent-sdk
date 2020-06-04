# this jq filter takes an array of api-server resource definition
# and flattens the version array of each resource
# ie a resource with two versions
# {"group": "management", "kind": "APIService", "spec": {"versions": [{"name": "v1alpha1", "schema": {}}, {"name": "v1alpha2",  "schema": {}}]}}
# becomes
# [{"group": "management", "kind": "APIService", "version": "v1alpha2", "schema": {}}, {"group": "management", "kind": "APIService", "version": "v1alpha2", "schema": {}}]

[.[] | . as $root | .spec.versions[] | . + {"group": $root.group, "kind": $root.kind, "scope": $root.scope, "version": .name, "names": $root.spec.names }]
