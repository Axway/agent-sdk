/**
 * Resources for the api server are fetched from the provided host and types are generated based on OAS doc retrieved.
 * Resources are split into "main resources" and "sub resources". Main resources are generated from the go template.
 * A resource and a client are generated together. This allows for an easy way to customize the behavior of the generated types.
 * Sub resources are generated from the openapi-generator and will be referenced by the main api server resources.
 *
 * A "main resource" would be something like APIService, SpecDiscovery or AWSDataPlane. Their files are named based on the name of the resource.
 * You will find the APIService type in the APIService.go file.
 *
 * A "sub resource" would be another type that the main resource depends on like APIServiceSpec or AWSDataPlaneSpec.
 * These files are generated from the openapi-generator and their name comes from the generator.
 * The APIServiceSpec type will be found in the model_api_service_spec.go file.
 *
 *
 */

const { execSync } = require("child_process");
const https = require("https");
const http = require("http");
const fs = require("fs");
const { exit } = require("process");

const outDir = process.env.OUTDIR;
const clientsPath = outDir + "/clients/";
const modelsPath = outDir + "/models/";
const resourcesTmplPath = "resources.tmpl";
const clientsTmplPath = "clients.tmpl";

const fetch = () => {
	const [, , protocol, host, port] = process.argv;
	if (!protocol || !host || !port) {
		throw new Error(
			"Protocol, host, and port are required. Ex: node generate.js https apicentral.axway.com 443",
			`protocol: ${protocol}, host: ${host}, port: ${port}`
		);
	}

	const options = {
		hostname: host,
		port: Number(port),
		path: "/apis/docs",
		method: "GET",
		headers: {
			"content-type": "application/json",
		},
	};

	return new Promise((resolve, reject) => {
		const h = protocol === "http" ? http : https;
		const req = h.request(options, (res) => {
			let body = "";
			res.on("data", (chunk) => {
				body += chunk;
			});
			res.on("end", () => {
				resolve(JSON.parse(body));
			});
		});
		req.on("error", (err) => reject(err));
		req.end();
	});
};

fetch()
	.then((resources) => {
		if (resources == "") {
			process.exit(1);
		}
		const [subResources, mainResources] = createMainAndSubResources(resources);
		// uncomment to see how the resources are grouped
		//fs.writeFileSync(outDir + '/sub-resources.json', JSON.stringify(subResources));
		//fs.writeFileSync(outDir + '/main-resources.json', JSON.stringify(mainResources));
		delete subResources.api; // the api resources are common resources, and have been written manually.
		writeSubResources(subResources);
		writeMainResources(mainResources);
		writeSet(mainResources);
	})
	.catch((err) => {
		console.log("ERROR: ", err);
		process.exit(1);
	});

// sub resources are grouped together into their corresponding group & version. For each version found in each group the openapi-generator will be used
// to create the resources. This allows us to split resources up into their logical groups and give them their own package.
// If there are two groups, such as "management" and "definitions", and each have one version, "v1alpha1", then the generator will
// run twice to create the resource in the appropriate package based on its group and version.
const writeSubResources = (subResources) => {
	for (let groupKey in subResources) {
		const groupObj = subResources[groupKey];
		for (let versionKey in groupObj) {
			const data = JSON.stringify(groupObj[versionKey]);
			const res = execSync(
				`openapi-generator-cli generate -g go -i /dev/stdin --package-name ${versionKey} --output ${modelsPath}${groupKey}/${versionKey} --global-property modelDocs=false,models << 'EOF'\n${data}\nEOF`
			);
		}
	}
};

const writeMainResources = (mainResources) => {
	for (let groupKey in mainResources) {
		const groupObj = mainResources[groupKey];

		for (let versionKey in groupObj) {
			const spec = groupObj[versionKey];
			const { schemas } = spec.components;

			for (let schemaKey in schemas) {
				const resource = createGomplateResource(schemas[schemaKey]);
				const { group, version, kind, fields } = resource;
				const file = `${group}/${version}/${kind}.go`;

				markEmptySpecs(fields);

				const input = `\'${JSON.stringify(resource)}\'`;
				// make the folders if they do not exist
				execSync(`mkdir -p ${clientsPath}${group}/${version}`).toString();
				execSync(`mkdir -p ${modelsPath}${group}/${version}`).toString();

				// create the models using the go template
				const model = `${modelsPath}${file}`;
				execSync(
					`echo ${input} | gomplate --context res="stdin:?type=application/json" -f ${resourcesTmplPath} --out "${model}"`
				);
				console.log(`Created model ${model}`);

				// creat the clients using the go template
				const client = `${clientsPath}${file}`;
				execSync(
					`echo ${input} | gomplate --context res="stdin:?type=application/json" -f ${clientsTmplPath} --out "${client}"`
				).toString();
				console.log(`Created client ${client}`);
			}
		}
	}
};

const createMainAndSubResources = (spec) => {
	// main resources are passed to the go template. subresources are passed to the openapi-generator
	const mainResources = {};
	const subResources = Object.keys(spec.components.schemas).reduce(
		(acc, schemaKey) => {
			addResourceToGroupVersion(
				spec.components.schemas[schemaKey]["x-axway-group"]
					? mainResources
					: acc,
				spec,
				schemaKey
			);
			return acc;
		},
		{}
	);
	return [subResources, mainResources];
};

const addResourceToGroupVersion = (acc, spec, schemaKey) => {
	const { schemas } = spec.components;
	const [group, version, kind] = schemaKey.split(".");
	// if the group does not exist, create the grouped resource
	if (!acc[group]) {
		acc[group] = {
			[version]: createOasSchema(spec.openapi, {
				[kind]: schemas[schemaKey],
			}),
		};
	}
	// if the group exists, but the version does not
	else if (acc[group] && !acc[group][version]) {
		acc[group][version] = createOasSchema(spec.openapi, {
			[kind]: schemas[schemaKey],
		});
	}
	// if the group and the version already exist
	else if (acc[group] && acc[group][version]) {
		acc[group][version].components.schemas = {
			...acc[group][version].components.schemas,
			[kind]: schemas[schemaKey],
		};
	}
};

// Gets the unique fields for the particular resource so that the resources.tmpl file can generate the fields
const filterFields = (resource) => {
	const { properties } = resource;
	const fields = {};

	const commonFields = new Set([
		"group",
		"apiVersion",
		"kind",
		"name",
		"title",
		"metadata",
		"finalizers",
		"attributes",
		"tags",
	]);

	for (key in properties) {
		if (!commonFields.has(key)) {
			fields[key] = properties[key];
		}
	}
	return fields;
};

// openapi-generator does not create a file when the resource is empty, like MeshSpec, which is an empty object with no keys.
// To check for this we must check if a file was generated. If we have a resource for something like MeshSpec, but no file generated then set the MeshSpec value to false.
const markEmptySpecs = (fields) => {
	for (key in fields) {
		const { $ref } = fields[key];
		const [, , , gvk] = $ref.split("/");
		const [group, version, kind] = gvk.split(".");
		try {
			let file =
				"model_" +
				kind
					.replace("API", "Api")
					.replace("AWS", "Aws")
					.replace(/([A-Z])/g, " $1")
					.trim()
					.replace(/\W/g, "_")
					.toLowerCase();
			fs.readFileSync(`${modelsPath}/${group}/${version}/${file}.go`);
			fields[key] = true;
		} catch (e) {
			if (e.code === "ENOENT") {
				fields[key] = false;
			} else {
				console.log("Error reading file: ", e);
			}
		}
	}
};

// Resources are grouped into their own mini oas spec so that the openapi-generator can be used to group related resources.
const createOasSchema = (openapi, schemas) => {
	return {
		openapi,
		paths: {},
		info: {
			title: "API Server specification.",
			version: "SNAPSHOT",
		},
		components: {
			schemas,
		},
	};
};

// The main API Server resources get passed into here, like APISpec, Environment, etc. Used to format the object before parsing it with the resources.tmpl file.
const createGomplateResource = (resource) => {
	let scopes = resource["x-axway-scopes"]
		? resource["x-axway-scopes"].map((scope) => scope.kind)
		: null;
	if (scopes) {
		scopes.sort();
	}
	return {
		group: resource["x-axway-group"],
		kind: resource["x-axway-kind"],
		version: resource["x-axway-version"],
		scoped: resource["x-axway-scoped"],
		scope: scopes ? scopes[0] : null, // temporarily pass the first scope in until the template can handle multiple scopes.
		scopes: scopes,
		resource: resource["x-axway-plural"],
		fields: filterFields(resource),
	};
};

// The clients Set is generated from all the main resources.
const writeSet = (resources) => {
	var setResources = [];
	Object.entries(resources).forEach(([group, versions]) => {
		Object.entries(versions).forEach(([version, versionFields]) => {
			kinds = Object.entries(versionFields.components.schemas).map(
				([kind, { "x-axway-scoped": scoped }]) => {
					return { kind, scoped };
				}
			);
			setResources.push({ group, version, kinds });
		});
	});
	const setInput = JSON.stringify({ set: setResources }, null, 2);

	execSync(
		`gomplate --context input='stdin:?type=application/json' -f ./set.tmpl --out "` +
			outDir +
			`/clients/set.go"`,
		{ input: setInput }
	);
};
