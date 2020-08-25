# API Server Resources

The code in this folder is generated from the yaml files that are pulled directly from the UnifiedCatalog repository.
To generate the models run `make unifiedcatalog-generate` from the root of the project (apigov/apic_agents_sdk).
The yaml is passed to openapi-generator, and ultimately the generated files are placed in apigov/apic_agents_sdk/pkg/apic/unifiedcatalog/models.

For information regarding openapi-generator, see https://github.com/OpenAPITools/openapi-generator

For information regarding yq, see https://github.com/mikefarah/yq

For information regarding gomplate, see https://github.com/hairyhenderson/gomplate


# Pre-requisites
In order to generate the code you need the following tools. Note that these instructions have been tested on linux. Mac instructions may vary.
1. install node: `sudo apt install nodejs`
2. install openapi-generator: `npm install @openapitools/openapi-generator-cli -g`
3. install yq: despite the instructions on the website (which may work), we did 
```
    wget https://github.com/mikefarah/yq/releases/download/3.3.2/yq_linux_amd64
    sudo chmod +x yq_linux_amd64
    sudo mv yq_linux_amd64 /usr/bin/yq
```
> Note that the scripts will NOT work for yq releases below 3.3.0
4. install gomplate: `go get github.com/hairyhenderson/gomplate/cmd/gomplate`


You should now be all set to run `make unifiedcatalog-generate`.

Good luck!!!