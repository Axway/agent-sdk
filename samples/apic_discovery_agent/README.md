
# Prerequisite
1. Golang 
2. Make

# Steps to implement discovery agent using this stub
1. Locate the commented tag "CHANGE_HERE" for package import paths in all files and fix them to reference the path correctly.
2. Run "make dep" to resolve the dependencies. This should resolve all dependency packages and vendor them under ./vendor directory
3. Update Makefile to change the name of generated binary image from *apic_discovery_agent* to the desired name. Locate *apic_discovery_agent* string and replace with desired name
4. Update pkg/cmd/root.go to change the name and description of the agent. Locate *apic_discovery_agent* and *Sample Discovery Agent* and replace to desired values
5. Update pkg/config/config.go to define the gateway specific configuration
    - Locate *gateway-section* and replace with the name of your gateway. Same string in pkg/cmd/root.go and sample YAML config file
    - Define gateway specific config properties in *GatewayConfig* struct. Locate the struct variables *ConfigKey1* & struct *config_key_1* and add/replace desired config properties
    - Add config validation. Locate *ValidateCfg()* method and update the implementation to add validation specific to gateway specific config.
    - Update the config binding with command line flags in init(). Locate *gateway-section.config_key_1* and add replace desired config property bindings
    - Update the initialization of gateway specific by parsing the binded properties. Locate *ConfigKey1* & *gateway-section.config_key_1* and add/replace desired config properties
6. Update pkg/gateway/client.go to implement the logic to discover and fetch the details related of the APIs.
    - Locate *DiscoverAPIs()* method and implement the logic
    - Locate *buildServiceBody()* method and update the Set*() method according to the API definition from gateway
7. Run "make build" to build the agent
8. Rename *apic_discovery_agent.yaml* file to the desired agents name and setup the agent config in the file.
9. Execute the agent by running the binary file generated under *bin* directory. The YAML config must be in the current working directory 

Reference: [SDK Documentation - Building Discovery Agent](https://github.com/Axway/agent-sdk/blob/main/docs/discovery/index.md)