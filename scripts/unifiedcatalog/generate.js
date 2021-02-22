const { execSync } = require('child_process');

const packagename = 'unifiedcatalog';
const stdin = './unifiedcatalog.json';
const group = 'models';
const version = 'v1';
//const proj="github.com/Axway/agent-sdk"

const getResourceFile = execSync(
	`curl https://apicentral.axway.com/api/unifiedCatalog/${version}/docs -o ${stdin} --silent`,
);

// install openapi-generator: `npm install @openapitools/openapi-generator-cli" -g`
// openapi generator 5.0.0+ refactored the go generation and it breaks some things, 4.3.1 verified working.
// openapi-generator-cli version-manager set 4.3.1
const generateFiles = execSync(
	`openapi-generator-cli generate -g go -i ${stdin} --package-name ${packagename} --output pkg/apic/${packagename}/${group}/ --global-property modelDocs=false --global-property models --global-property apiDocs=false`,
);

const removeResourceFile = execSync(`rm ${stdin}`);
