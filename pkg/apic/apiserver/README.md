# API Server Resources

The code in this folder is generated from the yaml files in the definitions folder. Thes yaml files come directly from the API Server.
To generate the models run `make apiserver_generate` from the root of the project.
The code in models/api is hand written code. The rest of the code is generated from the yaml in apiserver/defnitions and passed to openapi-generator

# Pre-requisites 
In order to generate the code you need the following tools.
1. npm i -g openapi-generator
2. pip install yq
3. sudo apt-get install jq OR brew install jq
4. GO111MODULE=yes go get github.com/hairyhenderson/gomplate/cmd/gomplate