# Building Discovery Agent

The Amplify Central Discovery Agents can be used for discovering APIs managed by external API Gateway and publish API Server resources to Amplify Central. The Amplify Agents SDK helps in building discovery agent by providing the necessary config, command line parser and interfaces to manage the communication with Amplify Central.

## Table of Contents

- [Central Configuration](#Central-Configuration)
- [Agent Configuration](#Agent-Configuration)
- [Setting up command line parser and binding agent config](#Setting-up-command-line-parser-and-binding-agent-config)
- [Filtering](#Filtering)
- [Processing Discovery](#Processing-Discovery)
- [Publishing changes to Central](#Publishing-changes-to-Central)
- [Subscriptions](./subscriptions.md)
- [Validating ConsumerInstance](#Validating-ConsumerInstance)
- [Building the Agent](#Building-the-Agent)
- [Executing Discovery Agent](#Executing-Discovery-Agent)

## Central Configuration

The Amplify Agents SDK provides a predefined configuration that can be set up based on yaml file, using environment variables or passed as command line flags. This configuration is used for setting up parameter that will be used for communicating with Amplify Central. In addition, it is also used to set up subscription processing, see [subscriptions](./subscriptions.md)

Below is the list of Central configuration properties in YAML and their corresponding environment variables that can be set to override the config in YAML.

| YAML property                  | Variable name                  | Description                                                                                                                                                                                                                                                                                                               |
|--------------------------------|--------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| central.mode                   | CENTRAL_MODE                   | Mode in which Agent operates to publish APIs to Central (`publishToEnvironment` = API Service, `publishToEnvironmentAndCatalog` = API Service and Catalog asset)                                                                                                                                                          |
| central.url                    | CENTRAL_URL                    | The URL to the Amplify Central instance being used for Agents (default value: US =  `<https://apicentral.axway.com>` / EU = `https://central.eu-fr.axway.com`)                                                                                                                                                            |
| central.organizationID         | CENTRAL_ORGANIZATIONID         | The Organization ID from Amplify Central. Locate this at Platform > User > Organization.                                                                                                                                                                                                                                  |
| central.team                   | CENTRAL_TEAM                   | The name of the team in Amplify Central that all APIs will be linked to. Locate this at Amplify Central > Access > Team Assets.(default to `Default Team`)                                                                                                                                                                |
| central.environment            | CENTRAL_ENVIRONMENT            | Name of the Amplify Central environment where API will be hosted.                                                                                                                                                                                                                                                         |
| central.additionalTags         | CENTRAL_ADDITIONALTAGS         | Additional tag names to publish while publishing the API. Helpful to identify the API source. It is a comma separated list.                                                                                                                                                                                               |
| central.auth.url               | CENTRAL_AUTH_URL               | The Amplify login URL: `<https://login.axway.com/auth>`                                                                                                                                                                                                                                                                   |
| central.auth.clientID          | CENTRAL_AUTH_CLIENTID          | The client identifier associated to the Service Account created in Amplify Central. Locate this at Amplify Central > Access > Service Accounts > client Id.                                                                                                                                                               |
| central.auth.privateKey        | CENTRAL_AUTH_PRIVATEKEY        | The private key associated with the Service Account.                                                                                                                                                                                                                                                                      |
| central.auth.publicKey         | CENTRAL_AUTH_PUBLICKEY         | The public key associated with the Service Account.                                                                                                                                                                                                                                                                       |
| central.auth.keyPassword       | CENTRAL_AUTH_KEYPASSWORD       | The password for the private key, if applicable.                                                                                                                                                                                                                                                                          |
| central.auth.timeout           | CENTRAL_AUTH_TIMEOUT           | The timeout to wait for the authentication server to respond (ns - default, us, ms, s, m, h). Set to 10s.                                                                                                                                                                                                                 |
| central.ssl.insecureSkipVerify | CENTRAL_SSL_INSECURESKIPVERIFY | Controls whether a client verifies the server's certificate chain and host name. If true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.                                                                       |
| central.ssl.cipherSuites       | CENTRAL_SSL_CIPHERSUITES       | An array of strings. It is a list of supported cipher suites for TLS versions up to TLS 1.2. If CipherSuites is nil, a default list of secure cipher suites is used, with a preference order based on hardware performance. See [Supported Cipher Suites](/docs/central/connect-api-manager/agent-security-api-manager/). |
| central.ssl.minVersion         | CENTRAL_SSL_MINVERSION         | String value for the minimum SSL/TLS version that is acceptable. If zero, empty TLS 1.0 is taken as the minimum. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3.                                                                                                                                                      |
| central.ssl.maxVersion         | CENTRAL_SSL_MAXVERSION         | String value for the maximum SSL/TLS version that is acceptable. If empty, then the maximum version supported by this package is used, which is currently TLS 1.3. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3.                                                                                                    |
| central.proxyURL               | CENTRAL_PROXYURL               | The URL for the proxy for Amplify Central `<http://username:password@hostname:port>`. If empty, no proxy is defined.                                                                                                                                                                                                      |

The following is a sample of Central configuration in YAML

```
central:
    url: https://apicentral.axway.com
    organizationID: "123456789"
    team: APIDev
    environment: remote-gw
    additionalTags: DiscoveredByCustomAgent
    auth:
        clientId: DOSA_3ecfferff6ab694badb1ba8e1cfb28f7u8
        privateKey: ./private_key.pem
        publicKey: ./public_key.pem
```

### Configuration interfaces

Amplify Agents SDK expose the following interfaces to retrieve the configuration items.

```
// Central Configuration
type CentralConfig interface {
 GetAgentType() AgentType
 IsPublishToEnvironmentMode() bool
 IsPublishToEnvironmentOnlyMode() bool
 IsPublishToEnvironmentAndCatalogMode() bool
 GetAgentMode() AgentMode
 GetAgentModeAsString() string
 GetTenantID() string
 GetEnvironmentID() string
 GetEnvironmentName() string
 GetTeamName() string
 GetTeamID() string

 GetURL() string
 GetPlatformURL() string
 GetCatalogItemsURL() string
 GetAPIServerURL() string
 GetEnvironmentURL() string
 GetServicesURL() string
 GetRevisionsURL() string
 GetInstancesURL() string
 DeleteServicesURL() string
 GetConsumerInstancesURL() string
 GetAPIServerSubscriptionDefinitionURL() string
 GetAPIServerWebhooksURL() string
 GetAPIServerSecretsURL() string

 GetSubscriptionURL() string
 GetSubscriptionConfig() SubscriptionConfig

 GetCatalogItemSubscriptionsURL(string) string
 GetCatalogItemSubscriptionStatesURL(string, string) string
 GetCatalogItemSubscriptionPropertiesURL(string, string) string
 GetCatalogItemSubscriptionDefinitionPropertiesURL(string) string
 GetCatalogItemByIDURL(catalogItemID string) string

 GetAuthConfig() AuthConfig
 GetTLSConfig() TLSConfig
 GetTagsToPublish() string

 GetProxyURL() string
 GetPollInterval() time.Duration 
}
```

```
// Central Authentication config
type AuthConfig interface {
 GetTokenURL() string
 GetRealm() string
 GetAudience() string
 GetClientID() string
 GetPrivateKey() string
 GetPublicKey() string
 GetKeyPassword() string
 GetTimeout() time.Duration
}
```

```
// TLS Config
type TLSConfig interface {
 GetNextProtos() []string
 IsInsecureSkipVerify() bool
 GetCipherSuites() []TLSCipherSuite
 GetMinVersion() TLSVersion
 GetMaxVersion() TLSVersion
 BuildTLSConfig() *tls.Config
}
```

## Agent Configuration

The agent can define a struct that holds the configuration it needs specifically to communicate with the external API Gateway. The agent config struct properties can be bound to command line processor to set up config, see [Setting up command line parser and binding agent config](#setting-up-command-line-parser-and-binding-agent-config)

### Sample Agent configuration definition

```
type AgentConfig struct {
 TenantID          string `config:"tenantID"`
 ClientID          string `config:"clientID"`
 ClientSecret      string `config:"clientSecret"`
 SubscriptionID    string `config:"subscriptionID"`
 ResourceGroupName string `config:"resourceGroupName"`
 ApimServiceName   string `config:"apimServiceName"`
 Filter            string `config:"filter"`
}
```

To validate the config, the following interface provided by config package in Amplify Agents SDK must be implemented for the agent config. The ValidateCfg() method is called by Amplify Agents SDK after parsing the config from command line.

```
// IConfigValidator - Interface to be implemented for config validation by agent
type IConfigValidator interface {
 ValidateCfg() error
}
```

For e.g.

```
// ValidateCfg - Validates the agent config
func (c *AgentConfig) ValidateCfg() (err error) {
 if c.TenantID == "" {
  return errors.New("Error: azure.tenantID is empty"))
 }

 if c.ClientID == "" {
  return errors.New("Error: azure.tenantID is empty"))
 }

 ...
 ...

 return nil
}

```

If there are ResourceInstance values that you want to apply to your agent config, the following interface provided by config package in Amplify Agents SDK must be implemented for the agent config. The ApplyResources() method is called by Amplify Agents SDK after parsing the config from command line.

```
// IResourceConfigCallback - Interface to be implemented by configs to apply API Server resource for agent
type IResourceConfigCallback interface {
 ApplyResources(agentResource *v1.ResourceInstance) error
}
```

For e.g.

```
// ApplyResources - Applies the agent and dataplane resource to config
func (a *AgentConfig) ApplyResources(agentResource *v1.ResourceInstance) error {
 var da *v1alpha1.DiscoveryAgent
 if agentResource.ResourceMeta.GroupKind.Kind == "DiscoveryAgent" {
  da = &v1alpha1.DiscoveryAgent{}
  err := da.FromInstance(agentResource)
  if err != nil {
   return err
  }
 }
 // copy any values from the agentResource to the AgentConfig
 ...
 ...

 return nil
}

```

### Sample Agent YAML configuration

```
central:
    url: https://apicentral.axway.com
    organizationID: "123456789"
    team: APIDev
    environment: remote-gw
    additionalTags: DiscoveredByCustomAgent
    auth:
        clientId: DOSA_3ecfferff6ab694badb1ba8e1cfb28f7u8
        privateKey: ./private_key.pem
        publicKey: ./public_key.pem

azure:
    tenantID: "az-tenant-id"
    clientID: "az-client-id"
    clientSecret: xxxxxx
    subscriptionID: "az-apim-subscription"
    resourceGroupName: "az-apim-rg"
    apimServiceName: "az-apim"
    filter: tag.Contains("somevalue")
```

## Setting up command line parser and binding agent config

Amplify Agents SDK internally uses [Cobra](https://github.com/spf13/cobra) for providing command line processing and [Viper](https://github.com/spf13/viper) to bind the configuration with command line processing and YAML based config file. The Amplify Agents SDK exposes an  interface for predefined configured root command for Agent that set up Central Configuration. The Agent root command allows to hook in the main routine for agent execution and a callback method that get called on initialization to set up agent specific config. The Amplify Agents SDK root command also allows the agent to set up command line flags and properties that are agent specific and bind these flag/properties to agent config.

### Sample of agent command initialization and agent config setup

```

// RootCmd - Agent root command
var RootCmd corecmd.AgentRootCmd
var azConfig *config.AzureConfig

func init() {
 // Create new root command with callbacks to initialize the agent config and command execution.
 // The first parameter identifies the name of the yaml file that agent will look for to load the config
 RootCmd = corecmd.NewRootCmd(
  "apic_discovery_agent_sample",
  "Sample Discovery Agent",  // you can set this to "" if you set BuildAgentDescription in your makefile
  initConfig,
  run,
  corecfg.DiscoveryAgent,
 )

 // Get the root command properties and bind the config property in YAML definition
 rootProps := RootCmd.GetProperties()
 rootProps.AddStringProperty("azure.tenantID", "", "Azure tenant ID")
 rootProps.AddStringProperty("azure.clientID", "", "Azure client ID")
 rootProps.AddStringProperty("azure.clientSecret", "", "Azure client secret")
 rootProps.AddStringProperty("azure.subscriptionID", "", "Azure subscription ID")
 rootProps.AddStringProperty("azure.resourceGroupName", "", "Azure resource group name")
 rootProps.AddStringProperty("azure.apimServiceName", "", "Azure API Management service name")
}

// Callback that agent will call to process the execution
func run() error {
 // Code for discovering API and publish
 return nil
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Amplify Agents SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {

 rootProps := RootCmd.GetProperties()
 // Parse the config from bound properties and set up agent config
 azConfig = &config.AzureConfig{
  TenantID:          rootProps.StringPropertyValue("azure.tenantID"),
  ClientID:          rootProps.StringPropertyValue("azure.clientID"),
  ClientSecret:      rootProps.StringPropertyValue("azure.clientSecret"),
  SubscriptionID:    rootProps.StringPropertyValue("azure.subscriptionID"),
  ResourceGroupName: rootProps.StringPropertyValue("azure.resourceGroupName"),
  ApimServiceName:   rootProps.StringPropertyValue("azure.apimServiceName"),
 }

 agentConfig := config.AgentConfig{
  CentralCfg: centralConfig,
  AzureCfg:   azConfig,
 }
 return agentConfig, nil
}

```

## Filtering

The Amplify Agents SDK provides github.com/Axway/agent-sdk/pkg/filter package to allow setting up config for filtering the discovered APIS for publishing them to Amplify Central. The filter expression to be evaluated for discovering the API from Axway Edge API Gateway. The filter value is a conditional expression that can use logical operators to compare two value.
The conditional expression must have "tag" as the prefix/selector in the symbol name. For e.g.

```
azure:
   filter: tag.SOME_TAG == "somevalue"
```

The expression can be a simple condition as shown above or compound condition in which more than one simple conditions are evaluated using logical operator.

For e.g.

```
azure:
   filter: tag.SOME_TAG == "somevalue" || tag.ANOTHER_TAG != "some_other_value"
```

In addition to logical expression, the filter can hold call based expressions. Below are the list of supported call expressions

### Exists

Exists call can be made to evaluate if the tag name exists in the list of tags on API. This call expression can be used as unary expression
For e.g.

```
azure:
   filter: tag.SOME_TAG.Exists()
```

### Any

Any call can be made in a simple expression to evaluate if the tag with any name has specified value or not in the list of tags on the API.
For e.g.

```
azure:
   filter: tag.Any() == "Tag with some value" || tag.Any() != "Tag with other value"
```

### Contains

Contains call can be made in a simple expression to evaluate if the the specified tag contains specified argument as value. This call expression requires string argument that will be used to perform lookup in tag value
For e.g.

```
tag.Contains("somevalue")
```

### MatchRegEx

MatchRegEx call can be used for evaluating the specified tag value to match specified regular expression. This call expression requires a regular expression as the argument.
For e.g.

```
tag.MatchRegEx("(some){1}")
```

## Processing Discovery

The agent can discover APIs in external API Gateway based on the capability it provides. This could be event based mechanism where config change from API gateway can be received or agent can query/poll for the API specification using the dataplane specific SDK. To process the discovery and publishing the definitions to Amplify Central the following properties are needed.

| API Service property | Description                                                                                                                                |
|----------------------|--------------------------------------------------------------------------------------------------------------------------------------------|
| ID                   | ID of the API.                                                                                                                             |
| PrimaryKey           | Optional PrimaryKey that will be used, in place of the ID, to identify APIs on the Gateway.                                                |
| Title                | Name of the API that will be used as Amplify Central Catalog name.                                                                         |
| Description:         | A brief summary about the API.                                                                                                             |
| Version:             | Version of the API.                                                                                                                        |
| URL:                 | Endpoint for the API service.                                                                                                              |
| Auth policy:         | Authentication/Authorization policies applied to API. For now, Amplify Central supports passthrough, api key and oauth.                    |
| Specification:       | The API service specification. The Amplify Agents SDK provides support for swagger 2, openapi 3, WSDL, Protobuf, AsyncAPI or Unstructured. |
| Documentation:       | Documentation for the API.                                                                                                                 |
| Tags:                | List of resource tags.                                                                                                                     |
| Image:               | Image for the API service.                                                                                                                 |
| Image content type:  | Content type of the Image associated with API service.                                                                                     |
| Resource type        | Specifies the API specification type ("swaggerv2", "oas2", "oas3", "wsdl", "protobuf", "asyncapi" or "unstructured").                      |
| State/Status         | State representation of API in external API Gateway(unpublished/published).                                                                |
| Attributes           | List of string key-value pairs that will be set on the resources created by the agent.                                                     |
| Endpoints            | List of endpoints(protocol, host, port, base path) to override the endpoints specified in spec definition.                                 |

To set these properties the Amplify Agents SDK provides a builder (ServiceBodyBuilder) that allows the agent implementation to create a service body definition that will be used for publishing the API definition to Amplify Central.

In case where the *SetResourceType* method is not explicitly invoked, the builder uses the spec content to discovers the type ("swaggerv2", "oas2", "oas3", "wsdl", "protobuf", "asyncapi" or "unstructured").

### Unstructured data additional properties

Along with the above properties the following properties are on the ServiceBodyBuilder for unstructured data only.

| API Service property    | Description                              | Default (not set)   |
|-------------------------|------------------------------------------|---------------------|
| UnstructuredAssetType   | Type of asset for the unstructured data. | Asset               |
| UnstructuredContentType | Content type for this data.              | parse based on spec |
| UnstructuredLabel       | Label to display int he catalog item.    | Asset               |
| UnstructuredFilename:   | Filename of the file to download.        | APIName             |

The builder will use these properties when set, or use the default if not.

### Sample of creating service body using the builder

```
func (a *AzureClient) buildServiceBody(azAPI apim.APIContract, apiSpec []byte) (apic.ServiceBody, error) {
 return apic.NewServiceBodyBuilder().
  SetID(*azAPI.ID).
  SetAPIName(*azAPI.Name).
  SetTitle(a.getAzureAPITitle(azAPI)).
  SetURL(a.getAzureAPIURL(azAPI)).
  SetDescription(a.getAzureAPIDescription(azAPI)).
  SetAPISpec(apiSpec).
  SetVersion(a.getAzureAPIVersion(azAPI)).
  SetAuthPolicy(a.getAzureAPIAuthPolicy(azAPI)).
  SetDocumentation(a.getAzureAPIDocumentation(azAPI)).
  SetResourceType(apic.Oas3).
  Build()
}

func (a *AzureClient) getAzureAPITitle(azAPI apim.APIContract) string {
 return fmt.Sprintf("%s (Azure)", *azAPI.Name)
}

func (a *AzureClient) getAzureAPIURL(azAPI apim.APIContract) string {
 return *azAPI.ServiceURL + "/" + *azAPI.Path
}

func (a *AzureClient) getAzureAPIDescription(azAPI apim.APIContract) string {
 // Update description/summary if one exists in the API
 description := "API From Azure API Management Service"
 if azAPI.Description != nil && *azAPI.Description != "" {
  description = *azAPI.Description
 }
 return description
}

func (a *AzureClient) getAzureAPIVersion(azAPI apim.APIContract) string {
 version := "0.0.0"
 if azAPI.APIVersion != nil {
  version = *azAPI.APIVersion
 }
 return version
}

func (a *AzureClient) getAzureAPIDocumentation(azAPI apim.APIContract) []byte {
 var documentation []byte
 if azAPI.APIVersionDescription != nil {
  documentation = []byte(*azAPI.APIVersionDescription)
 }
 return documentation
}

func (a *AzureClient) getAzureAPIAuthPolicy(azAPI apim.APIContract) string {
 apiAuthSetting := azAPI.AuthenticationSettings
 authType := apic.Passthrough
 if apiAuthSetting != nil && apiAuthSetting.OAuth2 != nil {
  authType = apic.Oauth
 }
 return authType
}

```

## Publishing changes to Central

The Agent can use the service body definition built by earlier set up and call the *PublishAPI* method in the *agent* package to publish the discovered API to Amplify Central. The method uses the service body to create following API server resources

- APIService: Resource representing an Amplify Central API Service.
- APIServiceRevision: Resource representing an Amplify Central API Service Revision
- APIServiceInstance: Resource representing the deployed instance of the revision
- ConsumerInstance: Represents the resources holding information about publishing assets to Amplify Unified Catalog

When *PublishAPI* is called for the first time for the discovered API, each of the above mentioned resources gets created with generated names. On subsequent calls to the method for the same discovered API, the *APIService* and *ConsumerInstance* resources are updated, while a new resource for *APIServiceRevision* is created to represent the updated revision of the API. For update, the *APIServiceInstance* resources is updated unless the endpoint in the service definitions are changed which triggers a creation of a new *APIServiceInstance* resource.

The *PublishAPI* method while creating/updating the API server resources set the following attributes.

- externalAPIID: Holds the ID of API discovered from remote API Gateway
- externalAPIName: Holds the name of the API discovered from remote API Gateway
- createdBy: Holds the name of the Agent creating the resource

### Sample of publishing API to Amplify Central

```
 serviceBody, err := buildServiceBody(azAPI, exportResponse.Body)
 ...
 err = agent.PublishAPI(serviceBody)
 if err != nil {
  log.Fatalf("Error in publishing API to Amplify Central: %s", err)
 }
```

### Sample of published API server resources

*Note:* Few details are removed/updated in the sample resource definitions below for simplicity.

```
---
group: management
apiVersion: v1alpha1
kind: APIService
name: 37260bb8-203b-11eb-bac3-3af9d38d3457
title: musicalinstrumentsapi-azure (Azure)
metadata:
  id: e4f4922a75742c4501759dee158c0114
  ...
attributes:
  createdBy: AzureDiscoveryAgent
  externalAPIID: ...DISCOVERED-API-ID...
  externalAPIName: musicalinstrumentsapi-azure
spec:
  description: This is a sample Musical Instruments API.

---

group: management
apiVersion: v1alpha1
kind: APIServiceRevision
name: 37260bb8-203b-11eb-bac3-3af9d38d3457.1
title: musicalinstrumentsapi-azure (Azure)
metadata:
  id: e4fcb2ab75906c6201759dee183500e8
  ...
attributes:
  createdBy: AzureDiscoveryAgent
  externalAPIID: ...DISCOVERED-API-ID...
  externalAPIName: musicalinstrumentsapi-azure
spec:
  apiService: 37260bb8-203b-11eb-bac3-3af9d38d3457
  definition:
    type: oas3
    value: ...BASE64 ENCODED API SPECIFICATION...

---

group: management
apiVersion: v1alpha1
kind: APIServiceInstance
name: 37260bb8-203b-11eb-bac3-3af9d38d3457.1
title: musicalinstrumentsapi-azure (Azure)
metadata:
  id: e4f4922a75742c4501759dee1aa80118
  ...
attributes:
  createdBy: AzureDiscoveryAgent
  externalAPIID: ...DISCOVERED-API-ID...
  externalAPIName: musicalinstrumentsapi-azure
spec:
  endpoint:
    - host: beano-demo.azure-api.net
      port: 443
      routing:
        basePath: /music/v2
      protocol: https
  apiServiceRevision: 37260bb8-203b-11eb-bac3-3af9d38d3457.1

---

group: management
apiVersion: v1alpha1
kind: ConsumerInstance
name: 37260bb8-203b-11eb-bac3-3af9d38d3457
title: musicalinstrumentsapi-azure (Azure)
metadata:
  id: e4fcb2ab75906c6201759dee1c2e00ed
  ...
attributes:
  createdBy: AzureDiscoveryAgent
  externalAPIID: ...DISCOVERED-API-ID...
  externalAPIName: musicalinstrumentsapi-azure
spec:
  name: musicalinstrumentsapi-azure (Azure)
  state: PUBLISHED
  status: PUBLISHED
  version: 0.0.1
  visibility: RESTRICTED
  description: This is a sample Musical Instruments API.
  subscription:
    autoSubscribe: false
    ...
  apiServiceInstance: 37260bb8-203b-11eb-bac3-3af9d38d3457.1

```

## Subscriptions

See [Subscriptions](./subscriptions.md)

## Validating ConsumerInstance

Amplify Central *ConsumerInstance* resources hold information about the assets published to Amplify Unified Catalog. In order to keep these assets in sync with the associated discovered API, a background job runs to validate each *ConsumerInstance*. If the resource is no longer valid, it is an indication that the API has likely been removed and the resource can be cleaned up.

It is the responsibility of the individual agent to determine the validity of the *ConsumerInstance*. The Amplify Agents SDK provides a mechanism for the discovery agent to register an API validator for this purpose. The agent implementation can call *RegisterAPIValidator* method in the *agent* package and provide a callback method. The Amplify Agents SDK will periodically call this method in the agent, thereby allowing the agent to determine the validity of the API and keep the resources in sync. Note that if an agent does not register an API validator, the consumer instance will never be validated and will always be considered synced with API. The ConsumerInstance resource will never be removed.

Below is the example of registering the API validator callback and sample validator implementation

```
func run() error {
 agent.RegisterAPIValidator(azAgent.validateAPI)
}

func (a *Agent) validateAPI(apiID, stageName string) bool {
 // Add validation here if the API should be marked as invalid
 return true
}
```

Returning true from the validator will indicate that the *ConsumerInstance* is still valid. The Amplify Agents SDK will not remove the resource. Returning false will indicate to the Amplify Agents SDK that the resource should be removed, thereby keeping the resources and the APIs in sync.

## Building the Agent

The agents are applications built using [Go programming language](https://golang.org/). Go is open source programming language that gets statically compiled and comes with a rich toolset to obtain packages and building executables. The Amplify Agents SDK uses the Go module as the dependency management which was introduced in Go 1.11. Go modules is collection of packages with go.mod file in its root directory which defines the modules source paths used in the packages as imports.

The *go mod tidy* command will prune any unused dependencies from your *go.mod* and update the files to include used dependencies. The *go mod verify* command checks the dependencies, downloads them from the source repository and updates the cryptographic hashes in your go.sum file.

Run the following commands to resolve the dependencies

```
go mod tidy
go mod verify
```

After resolving the dependencies, run *make build* to compile the source and generate the binary executable for the target system.
The Amplify Agents SDK provides support for specifying the version of the agent at the build time. The following variables can be set by compile flags to set up agent name, version, commit SHA and build time.

- github.com/Axway/agent-sdk/pkg/cmd.BuildTime
- github.com/Axway/agent-sdk/pkg/cmd.BuildVersion
- github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha
- github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName - this is an internal name
- github.com/Axway/agent-sdk/pkg/cmd.BuildAgentDescription - this is a friendly description that will be displayed in the --version and --help commands
- github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion

The following is an example of the build command that can be configured in the Makefile

```
@export time=`date +%Y%m%d%H%M%S` && \
export version=`cat version` && \
export commit_id=`git rev-parse --short HEAD` && \
export sdk_version=`go list -m github.com/Axway/agent-sdk | awk '{print $$2}' | awk -F'-' '{print substr($$1, 2)}'` && \
go build -tags static_all \
	-ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=$${time}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=$${version}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=$${commit_id}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=SampleDiscoveryAgent' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentDescription=Sample Discovery Agent' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion=$${sdk_version}'" \
	-a -o ${WORKSPACE}/bin/discovery_agent ${WORKSPACE}/main.go
```

### Pre-requisites for executing the agent

- An Axway Amplify Central subscription in the Amplify™ platform. See [Get started with Amplify Central](https://axway-open-docs.netlify.app/docs/central/quickstart).

- An Amplify Central Service Account. See [Create a service account](https://axway-open-docs.netlify.app/docs/central/cli_central/cli_install/#22-create-a-service-account-using-the-amplify-central-ui).
- An Amplify Central environment. See [Create environment](https://axway-open-docs.netlify.app/docs/central/cli_central/cli_environments/#create-an-environment).

## Executing Discovery Agent

The Agent built using Amplify Agents SDK can be executed by running the executable. The agent on initialization tries to load the configuration from following sources and applies the configuration properties in the order described below.

1. The configuration YAML file in the current working directory.
2. Environment variable defined for configuration override on the shell executing the agent
3. A file containing the list of environment variable override
4. Command line flags

Below is the sample for executing the agent when the config YAML file only is used.

```
cd <path-to-agent-install-directory>
./discovery_agent
```

Typically, the configuration YAML can be placed in the same directory as the agent executable, but alternatively the YAML file could be placed in another directory and then *pathConfig* command line flags can be used to specify the directory path containing the YAML file.

```
<path-to-agent-install-directory>/discovery_agent --pathConfig <directory-path-for-agent-yaml-config-file>
```

The following is an example of command to execute the agent with a file holding environment variables

```
cd <path-to-agent-install-directory>
./discovery_agent --envFile <path-of-env-file>/config.env
```

The agent configuration can also be passed as command line flags. Below is an example of agent usage that details the command line flags and configuration properties

```
cd <path-to-agent-install-directory>
./discovery_agent --help
Discovery Agent

Usage:
  discovery_agent [flags]

Flags:
      --envFile string                                                         Path of the file with environment variables to override configuration
  -h, --help                                                                   help for azure_discovery_agent
      --pathConfig string                                                      Configuration file path for the agent
      --status                                                                 Get the status of all the Health Checks
      --synchronize                                                            Run the sync process for the discovery agent
  -v, --version                                                                version for azure_discovery_agent
```
