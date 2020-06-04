# this jq filter takes a flattened resource array
# groups the entries in the array by group and version
# and for each group/version creates an openapi definition file
# containing all the spec and subresource definitions of the resources in the array

group_by({group,version}) | map(
                                {
                                  "group": (.[0].group),
                                  "version": (.[0].version),
                                  "openapi": {
                                     "components": {
                                         "schemas": ({}
																							+ (map(. as $root | .subresources |  if . then with_entries({"key": "\($root.kind)\(.key | ascii_upcase)", "value": .value.openAPIV3Schema}) else {} end) | reduce .[] as $x ({}; . + $x))
																							+  (map({"key": "\(.kind)SPEC", "value": .schema.openAPIV3Schema }) | from_entries))
                                      },
                                   "paths": {},
                                   "openapi" : "3.0.2",
                                   "info" : {
                                      "title" : "API Server specification.",
                                      "version": "SNAPSHOT"
}}})
