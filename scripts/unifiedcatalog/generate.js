const {execSync} = require("child_process");

const packagename = "unifiedcatalog";
const stdin = "./unifiedcatalog.json";
const group = "models";
const version = "v1";
//const proj="github.com/Axway/agent-sdk"

const getResourceFile = execSync(
    `curl https://apicentral.axway.com/api/unifiedCatalog/${version}/docs -o ${stdin} --silent`,
);

// Receiving a warning in the cli...
// [DEPRECATED] -D arguments after 'generate' are application arguments and not Java System Properties,
// please consider changing to --global-property, apply your system properties to JAVA_OPTS,
// or move the -D arguments before the jar option.
// changed -Dmodels to: --global-property models
const generateFiles = execSync(
    `openapi-generator generate -g go -i ${stdin} --package-name ${packagename} --output pkg/apic/${packagename}/${group}/ --global-property modelDocs=false --global-property models --global-property apiDocs=false`,
);

const removeResourceFile = execSync(`rm ${stdin}`);
