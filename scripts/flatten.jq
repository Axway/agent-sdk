.[] |= . as $root | .spec.versions[] | . + {"group": $root.group, "kind": $root.kind, "scope": $root.scope, "version": .name, "names": $root.spec.names }
