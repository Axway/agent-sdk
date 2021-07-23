# API Server Resources

The code in this folder is generated from the API Server OAS documentation (https://apicentral.axway.com/apis/docs)
To generate the models run `make apiserver-generate $PROTOCOL $HOST $PORT` from the root of the project (apigov/apic_agents_sdk).

For information regarding openapi-generator, see https://github.com/OpenAPITools/openapi-generator

For information regarding gomplate, see https://github.com/hairyhenderson/gomplate

# Pre-requisites
In order to generate the code you need the following tools. Note that these instructions have been tested on linux. Mac instructions may vary.
1. install node: `sudo apt install nodejs`
2. install openapi-generator: `npm install @openapitools/openapi-generator-cli -g`
// openapi generator 5.0.0+ refactored the Go generation and it breaks some things, 4.3.1 verified working.
3. openapi-generator-cli version-manager set 4.3.1
4. install gomplate: `GO111MODULE=yes go get github.com/hairyhenderson/gomplate/cmd/gomplate`

You should now be all set to run `make apiserver-generate $PROTOCOL $HOST $PORT`. ex: `make apiserver-generate https apicentral.axway.com 443`
