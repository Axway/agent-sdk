# Building Traceability Agent

The AMPLIFY Central Traceability Agents can be used for monitoring the traffic for APIs that were discovered by AMPLIFY Central Discovery Agent and publishing the traffic event to AMPLIFY Central API Observer. The Agent SDK helps in building custom elastic beat as AMPLIFY Central traceability agent by providing the necessary config, supported transports and interfaces to manage the communication with AMPLIFY Central. 

### Central Configuration
The SDK provides a predefined configuration that can be setup based on yaml file, using environment variables or passed as command line option. This configuration is used for setting up parameter that will be used for communicating with AMPLIFY Central. 

Below is the list of Central configuration properties in YAML and their corresponding environment variables that can be set to override the config in YAML.


| YAML propery                   | Variable name                  | Description                                                                                                                                                              |
|--------------------------------|--------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| central.mode                   | CENTRAL_MODE                   | Mode in which Agent operates to publish APIs to Central (`publishToEnvironment` = API Service, `publishToEnvironmentAndCatalog` = API Service and Catalog asset)         |
| central.url                    | CENTRAL_URL                    | The URL to the AMPLIFY Central instance being used for Agents (default value: US =  `<https://apicentral.axway.com>` / EU = `https://central.eu-fr.axway.com`)           |
| central.organizationID         | CENTRAL_ORGANIZATIONID         | The Organization ID from AMPLIFY Central. Locate this at Platform > User > Organization.                                                                                 |
| central.team                   | CENTRAL_TEAM                   | The name of the team in AMPLIFY Central that all APIs will be linked to. Locate this at AMPLIFY Central > Access > Team Assets.(default to `Default Team`)               |
| central.environment            | CENTRAL_ENVIRONMENT            | Name of the AMPLIFY Central environment where API will be hosted.                                                                                                        |
| central.deployment             | CENTRAL_DEPLOYMENT             | Specifies the AMPLIFY Central deployment. This could be "prod" or "prod-eu" based on the AMPLIFY Central region.                                                         |
| central.additionalTags         | CENTRAL_ADDITIONALTAGS         | Additional tag names to publish while publishing the API. Could help to identified the API source. It is a comma separated list.                                         |
| central.auth.url               | CENTRAL_AUTH_URL               | The AMPLIFY login URL: `<https://login.axway.com/auth>`                                                                                                                  |
| central.auth.clientID          | CENTRAL_AUTH_CLIENTID          | The client identifier associated to the Service Account created in AMPLIFY Central. Locate this at AMPLIFY Central > Access > Service Accounts > client Id.              |
| central.auth.privateKey        | CENTRAL_AUTH_PRIVATEKEY        | The private key associated with the Service Account.                                                                                                                     |
| central.auth.publicKey         | CENTRAL_AUTH_PUBLICKEY         | The public key associated with the Service Account.                                                                                                                      |
| central.auth.keyPassword       | CENTRAL_AUTH_KEYPASSWORD       | The password for the private key, if applicable.                                                                                                                         |
| central.auth.timeout           | CENTRAL_AUTH_TIMEOUT           | The timeout to wait for the authentication server to respond (ns - default, us, ms, s, m, h). Set to 10s.                                                                |
| central.ssl.insecureSkipVerify | CENTRAL_SSL_INSECURESKIPVERIFY | Controls whether a client verifies the server's certificate chain and host name. If true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.                                                                         |
| central.ssl.cipherSuites       | CENTRAL_SSL_CIPHERSUITES       | An array of strings. It is a list of supported cipher suites for TLS versions up to TLS 1.2. If CipherSuites is nil, a default list of secure cipher suites is used, with a preference order based on hardware performance. See [Supported Cipher Suites](/docs/central/connect-api-manager/agent-security-api-manager/).   |
| central.ssl.minVersion         | CENTRAL_SSL_MINVERSION         | String value for the minimum SSL/TLS version that is acceptable. If zero, empty TLS 1.0 is taken as the minimum. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3.     |
| central.ssl.maxVersion         | CENTRAL_SSL_MAXVERSION         | String value for the maximum SSL/TLS version that is acceptable. If empty, then the maximum version supported by this package is used, which is currently TLS 1.3. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3.                                                                                                      |
| central.proxyURL               | CENTRAL_PROXYURL               | The URL for the proxy for Amplify Central `<http://username:password@hostname:port>`. If empty, no proxy is defined.                                                     |

The following is a sample of Central configuration in YAML
```
central:
    url: https://apicentral.axway.com
    organizationID: "123456789"
    team: APIDev
    environment: remote-gw
	deployment: prod
    additionalTags: DiscoveredByCustomAgent
    auth:
        clientId: DOSA_3ecfferff6ab694badb1ba8e1cfb28f7u8
        privateKey: ./private_key.pem
        publicKey: ./public_key.pem
```

#### Configration interfaces
Agent SDK expose the following interfaces to retrieve the configuration items.

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

### Agent Configuration
TBD

### Initializing Agent and Ingestion service transport
TBD

### Transaction Event processing and Event Generation
TBD

### Building the Agent
TBD

### Executing Traceability Agent
TBD