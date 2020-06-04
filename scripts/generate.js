const { execSync } = require('child_process');
/**
 Takes an array of api-server resource definition and flattens the versions array of each resource
 {
 	"group": "management",
 	"kind": "APIService",
 	"spec": {
 			"versions": [ { "name": "v1alpha1", "schema": {} }, { "name": "v1alpha2", "schema": {} } ]
 	}
 }
 which becomes
 [
 	{ "group": "management", "kind": "APIService", "version": "v1alpha1", "schema": {} },
 	{ "group": "management", "kind": "APIService", "version": "v1alpha2","schema": {} }
 ]

 This gets passed into the openapi-generator command to create go code
*/

const newGroupedResource = (group, version) => ({
  group,
  version,
  openapi: {
    components: {
      schemas: {},
    },
    paths: {},
    openapi: '3.0.2',
    info: {
      title: 'API Server specification.',
      version: 'SNAPSHOT',
    },
  },
});

const isSpecEmpty = (spec) =>
  JSON.stringify(spec) === JSON.stringify({ type: 'object', additionalProperties: false });

// Convert the yaml definitions to an array of json objects.
const resources = JSON.parse(
  execSync(`cat pkg/apic/apiserver/definitions/*.yaml | yq -s .`).toString()
)
  // Pull each version out and add the group, kind, & scope values
  .map((resource) =>
    resource.spec.versions.map((version) => ({
      ...version,
      group: resource.group,
      kind: resource.kind,
      scope: resource.scope,
      version: version.name,
      names: resource.spec.names,
    }))
  )
  // Since versions is an array, each element is an array of objects. Flatten the 2-d array
  .reduce((acc, value) => acc.concat(value));

// Create initial grouped resources with no schemas defined.
// Grouped Resources contain a schema with all resources that belong to a resource identified by its group & kind
const initGroupedResources = resources.reduce((acc, resource) => {
  if (acc.length === 0) {
    acc.push(newGroupedResource(resource.group, resource.version));
  } else {
    let isResourceGroupFound = acc.some(
      (obj) => obj.group === resource.group && obj.version === resource.version
    );
    if (!isResourceGroupFound) {
      acc.push(newGroupedResource(resource.group, resource.version));
    }
  }

  return acc;
}, []);

// Add all resources to a GroupedResource's schema by its group & kind values.
const groupedResources = resources.reduce((acc, currentResource) => {
  const { group, kind, schema, subresources, version } = currentResource;
  const groupedResource = acc.find(
    (resource) => resource.group === group && resource.version === version
  );

  if (schema && schema.openAPIV3Schema && schema.openAPIV3Schema.constructor === Object) {
    groupedResource.openapi.components.schemas = {
      ...groupedResource.openapi.components.schemas,
      [`${kind}SPEC`]: schema.openAPIV3Schema,
    };
  } else {
    console.error('ERROR: schema or schema.openAPIV3Schema is not an object', schema);
  }
  if (subresources && subresources.constructor === Object) {
    const subresourceKeys = Object.entries(subresources).map(([key, value]) => [
      `${kind}${key.toUpperCase()}`,
      value.openAPIV3Schema,
    ]);
    groupedResource.openapi.components.schemas = {
      ...groupedResource.openapi.components.schemas,
      ...subresourceKeys.reduce((acc, value) => {
        acc[value[0]] = value[1];
        return acc;
      }, {}),
    };
  }
  return acc;
}, initGroupedResources);

for (groupedResource of groupedResources) {
  const { group, openapi, version } = groupedResource;
  const res = execSync(
    `openapi-generator generate -g go -i /dev/stdin --package-name ${version} --output pkg/apic/apiserver/models/${group}/${version} -DmodelDocs=false -Dmodels << 'EOF'\n${JSON.stringify(
      openapi
    )}\nEOF`
  );
  console.log(res.toString());
}

const gomplateResources = resources.map((resource) => ({
  kind: resource.kind,
  version: resource.version,
  group: resource.group,
  scope: resource.scope || null,
  resource: resource.names.plural,
  fields: {
    spec: !isSpecEmpty(resource.schema.openAPIV3Schema),
    ...Object.entries(resource.subresources || {})
      .map(([key, value]) => [key, !isSpecEmpty(value.openAPIV3Schema)])
      .reduce((acc, value) => {
        acc[value[0]] = value[1];
        return acc;
      }, {}),
  },
}));

// The main struct for each resources is generated via gomplate
for (resource of gomplateResources) {
  const input = `\'${JSON.stringify(resource)}\'`;
  execSync(
    `echo ${input} | gomplate --context res="stdin:?type=application/json" -f scripts/resources.tmpl --out "pkg/apic/apiserver/models/${resource.group}/${resource.version}/${resource.kind}.go"`
  );
  console.log(`Created ${resource.group}/${resource.version}/${resource.kind}.go`);
}
