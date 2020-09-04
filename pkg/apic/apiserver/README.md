# API Server Resources

The code in this folder is generated from the yaml files in the definitions folder. Thes yaml files have been copied (manually) directly from the API Server repository.
To generate the models run `make apiserver-generate` from the root of the project (apigov/apic_agents_sdk).
The code in models/api is hand written code. The rest of the code is generated from the yaml in apiserver/defnitions and passed to openapi-generator, which ultimately puts the generated models in apigov/apic_agent_sdk/pkg/apiserver/models/definitions(management)/v1alpha1.

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
>Note that the scripts will NOT work for yq releases below 3.3.0
4. install gomplate: `go get github.com/hairyhenderson/gomplate/cmd/gomplate`


You should now be all set to run `make apiserver-generate`.

Good luck!!!