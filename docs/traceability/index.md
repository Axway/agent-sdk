# Building Traceability Agent

The Amplify Central Traceability Agents can be used for monitoring the traffic for APIs that were discovered by Amplify Central Discovery Agent and publishing the traffic event to Amplify Central API Observer. The Agent SDK helps in building custom elastic beat as Amplify Central traceability agent by providing the necessary config, supported transports and interfaces to manage the communication with Amplify Central.

The traceability agents are custom elastic beats which generally has two main components
- a component that collects data 
- a component that publishes the data to specified output

The Agents SDK provides implementation for component to publish the data to Amplify ingestion service either using lumberjack or HTTP protocol. The agent developers can implement the component to collect the data or use existing beat implementations (e.g. filebeat) to collect the data. 

To ingest the traffic related events for the Amplify Central Observer, the event is required to be in a specific structure. The Agent SDK provides definition for the log event (transaction.LogEvent) that can be used to setup the event data required by Amplify Central Observer service. The log event can be either of summary or transaction type. Refer to section [](#log_event_format) for the details. The agent developer can choose to implement the mapping from the collected data to log event required by Amplify Central Observer service either while the data is collected by the custom beat logic or by setup output event processor to perform the mapping. The output event processor are invoked when the publisher is processing the event to be published over specified transport.

The Amplify ingestion service authenticates the publish request using the token issued by AxwayID. For the lumberjack protocol the token is required as a field in event getting published. With HTTP, the ingestion service authenticates the request using bearer token in "Authorization" header.

The Agent SDK provides a component for generating beat event from the mapped log event. This component take care of setting up the beat event with fields required by Amplify Central Observer service.

### Central Configuration
The SDK provides a predefined configuration that can be setup based on yaml file, using environment variables or passed as command line option. This configuration is used for setting up parameter that will be used for communicating with Amplify Central. 

Below is the list of Central configuration properties in YAML and their corresponding environment variables that can be set to override the config in YAML.

| YAML propery                     | Variable name                    | Description                                                                                                                                                                                                                                                                                                               |
|----------------------------------|----------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| central.url                      | CENTRAL_URL                      | The URL to the Amplify Central instance being used for Agents (default value: US =  `<https://apicentral.axway.com>` / EU = `https://central.eu-fr.axway.com`)                                                                                                                                                            |
| central.organizationID           | CENTRAL_ORGANIZATIONID           | The Organization ID from Amplify Central. Locate this at Platform > User > Organization.                                                                                                                                                                                                                                  |
| central.team                     | CENTRAL_TEAM                     | The name of the team in Amplify Central that all APIs will be linked to. Locate this at Amplify Central > Access > Team Assets.(default to `Default Team`)                                                                                                                                                                |
| central.environment              | CENTRAL_ENVIRONMENT              | Name of the Amplify Central environment where API will be hosted.                                                                                                                                                                                                                                                         |
| central.deployment               | CENTRAL_DEPLOYMENT               | Specifies the Amplify Central deployment. This could be "prod" or "prod-eu" based on the Amplify Central region.                                                                                                                                                                                                          |
| central.agentName                | CENTRAL_AGENTNAME                | The agent name of this agent on Amplify Central                                                                                                                                                                                                                                                                           |
| central.platformEnvironmentId    | CENTRAL_PLATFORMENVIRONMENTID    | The ID of the environment on Amplify to report usage to                                                                                                                                                                                                                                                                   |
| central.lighthouseURL            | CENTRAL_LIGHTHOUSEURL            | The Lighthouse URL the agent publishes usage reports                                                                                                                                                                                                                                                                      |
| central.publishUsage             | CENTRAL_PUBLISHUSAGE             | Enables/disables the sending of usage events to Amplify                                                                                                                                                                                                                                                                   |
| central.publishTraffic           | CENTRAL_PUBLISHTRAFFIC           | Enables/disabled the sending of traffic events to Amplify Central                                                                                                                                                                                                                                                         |
| central.eventAggregationInterval | CENTRAL_EVENTAGGREGATIONINTERVAL | The frequency in which the agent reports API usage to Amplify                                                                                                                                                                                                                                                             |
| central.reportActivityFrequency  | CENTRAL_REPORTACTIVITYFREQUENCY  | The frequency in which the agent published activity to Amplify Central                                                                                                                                                                                                                                                    |
| central.auth.url                 | CENTRAL_AUTH_URL                 | The Amplify login URL: `<https://login.axway.com/auth>`                                                                                                                                                                                                                                                                   |
| central.auth.clientID            | CENTRAL_AUTH_CLIENTID            | The client identifier associated to the Service Account created in Amplify Central. Locate this at Amplify Central > Access > Service Accounts > client Id.                                                                                                                                                               |
| central.auth.privateKey          | CENTRAL_AUTH_PRIVATEKEY          | The private key associated with the Service Account.                                                                                                                                                                                                                                                                      |
| central.auth.publicKey           | CENTRAL_AUTH_PUBLICKEY           | The public key associated with the Service Account.                                                                                                                                                                                                                                                                       |
| central.auth.keyPassword         | CENTRAL_AUTH_KEYPASSWORD         | The password for the private key, if applicable.                                                                                                                                                                                                                                                                          |
| central.auth.timeout             | CENTRAL_AUTH_TIMEOUT             | The timeout to wait for the authentication server to respond (ns - default, us, ms, s, m, h). Set to 10s.                                                                                                                                                                                                                 |
| central.ssl.insecureSkipVerify   | CENTRAL_SSL_INSECURESKIPVERIFY   | Controls whether a client verifies the server's certificate chain and host name. If true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.                                                                       |
| central.ssl.cipherSuites         | CENTRAL_SSL_CIPHERSUITES         | An array of strings. It is a list of supported cipher suites for TLS versions up to TLS 1.2. If CipherSuites is nil, a default list of secure cipher suites is used, with a preference order based on hardware performance. See [Supported Cipher Suites](/docs/central/connect-api-manager/agent-security-api-manager/). |
| central.ssl.minVersion           | CENTRAL_SSL_MINVERSION           | String value for the minimum SSL/TLS version that is acceptable. If zero, empty TLS 1.0 is taken as the minimum. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3.                                                                                                                                                      |
| central.ssl.maxVersion           | CENTRAL_SSL_MAXVERSION           | String value for the maximum SSL/TLS version that is acceptable. If empty, then the maximum version supported by this package is used, which is currently TLS 1.3. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3.                                                                                                    |
| central.proxyURL                 | CENTRAL_PROXYURL                 | The URL for the proxy for Amplify Central `<http://username:password@hostname:port>`. If empty, no proxy is defined.                                                                                                                                                                                                      |

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

#### Configuration interfaces
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

### Agent Specific Configuration
The agent can define a struct that holds the configuration it needs specifically to communicate with the external API Gateway. The agent config struct properties can be bound to command line processor to setup config, see [Setting up command line parser and binding agent config](#setting-up-command-line-parser-and-binding-agent-config)


#### Sample Agent specific configuration definition
```
type GatewayConfig struct {
	TrafficLogFilePath       string `config:"trafficLogFilePath"`
	ExcludeHeaders    []string `config:"excludeHeaders"`
	...
}
```

To validate the config, following interface provided by config package in SDK can be implemented for the agent config. The ValidateCfg() method is called by SDK after parsing the config from command line.
```
// IConfigValidator - Interface to be implemented for config validation by agent
type IConfigValidator interface {
	ValidateCfg() error
}
```

For e.g.

```
// ValidateCfg - Validates the agent config
func (c *GatewayConfig) ValidateCfg() (err error) {
	if c.LogFilePath == "" {
		return errors.New("Error: gateway.trafficLogFilePath is empty"))
	}

	...
	...

	return nil
}

```

### Amplify Ingestion output configuration

The SDK provides a predefined configuration for setting up the output transport the agent is going to use for publishing the events.

Below is the list of traceability output transport configuration properties in YAML and their corresponding environment variables that can be set to override the config in YAML.

| YAML propery                          | Variable name                 | Description                                                                                                                                           |
|---------------------------------------|-------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------|
| output.traceability.enabled           |                               | Flag for enabling traceability output                                                                                                                 |
| output.traceability.hosts             | TRACEABILITY_HOST             | The host name of the ingestion service (default: `ingestion-lumberjack.datasearch.axway.com:453`)                                                     |
| output.traceability.protocol          | TRACEABILITY_PROTOCOL         | The transport protocol to be used 'tcp' for lumberjack or 'https' for HTTPS protocol (default: `tcp`)                                                 |
| output.traceability.compression_level | TRACEABILITY_COMPRESSIONLEVEL | Specifies the gzip compression level (default: `3`)                                                                                                   |
| output.traceability.bulk_max_size     | TRACEABILITY_BULKMAXSIZE      | The max number of evnets in a single request to Condor (default: `100`)                                                                               |
| output.traceability.timeout           | TRACEABILITY_TIMEOUT          | Number of seconds to wait for responses from Condo (default: `300s`)                                                                                  |
| output.traceability.pipelining        | TRACEABILITY_PIPELINING       | Number of asynchronous batches to send, must be 0                                                                                                     |
| output.traceability.worker            | TRACEABILITY_WORKER           | Number of workers communicating with Condor (default: `2`)                                                                                            |
| output.traceability.slow_start        | TRACEABILITY_SLOWSTART        | When set to true the number of events sent to Condor is ramped up to bulk_max_size (default: `true`)                                                  |
| output.traceability.ssl.enable        | TRACEABILITY_ENABLE_SSL       | Enables SSL connections to Condor (default: `true`)                                                                                                   |
| output.traceability.proxy_url         | TRACEABILITY_PROXYURL         | The URL for the HTTP or SOCK5 proxy for Amplify ingestion service for e.g. `<http://username:password@hostname:port>`. If empty, no proxy is defined. |
| output.traceability.redaction         |                               | Refer to [Traceability redaction](#traceability-redaction)                                                                                            |

#### Sample Agent YAML configuration 

```
apic_traceability_agent:
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

  gateway:
    trafficLogFilePath: "./logs/traffic.log"
    excludeHeaders: ["x-header-1", "x-header-2"]
    ...
    ...

# Amplify Ingestion service
output.traceability:
  enabled: true
  hosts:
   - ${TRACEABILITY_HOST:"ingestion-lumberjack.datasearch.axway.com:453"}
  protocol: ${TRACEABILITY_PROTOCOL:"tcp"}
  compression_level: ${TRACEABILITY_COMPRESSIONLEVEL:3}
  ssl:
    enabled: true
    verification_mode: none
    cipher_suites:
      - "ECDHE-ECDSA-AES-128-GCM-SHA256"
      - "ECDHE-ECDSA-AES-256-GCM-SHA384"
      - "ECDHE-ECDSA-CHACHA20-POLY1305"
      - "ECDHE-RSA-AES-128-CBC-SHA256"
      - "ECDHE-RSA-AES-128-GCM-SHA256"
      - "ECDHE-RSA-AES-256-GCM-SHA384"
      - "ECDHE-RSA-CHACHA20-POLY1205"
  worker: 1
  pipelining: 0
  proxy_url: ${TRACEABILITY_PROXYURL:""}

```

### Setting up command line parser and binding agent config

Agent SDK internally uses [Cobra](https://github.com/spf13/cobra) for providing command line processing and [Viper](https://github.com/spf13/viper) to bind the configuration with command line processing and YAML based config file. The Agent SDK exposes an  interface for predefined configured root command for Agent that setup Central Configuration. The Agent root command allows to hook in the main routine for agent execution and a callback method that get called on initialization to setup agent specific config. The Agent SDK root command also allows the agent to setup command line flags and properties that are agent specific and bind these flag/properties to agent config.

As traceability agents are custom elastic beat, the agent root command can be setup by wrapping beat root command which sets up the command flags/properties and command execution required by elastic beat.

#### Sample of agent command initialization and agent config setup

```
// RootCmd - Agent root command
var RootCmd corecmd.AgentRootCmd
var beatCmd *libbeatcmd.BeatsRootCmd

func init() {
	name := "apic_traceability_agent"
	settings := instance.Settings{
		Name:          name,
		HasDashboards: true,
		ConfigOverrides: corecfg.LogConfigOverrides(),
	}

	// Initialize the beat command
	beatCmd = libbeatcmd.GenRootCmdWithSettings(beater.New, settings)
	cmd := beatCmd.Command
	// Wrap the beat command with the agent command processor with callbacks to initialize the agent config and command execution.
	RootCmd = corecmd.NewCmd(
		&cmd,
		name,                        // Name of the agent and yaml config file
		"Sample Traceability Agent", // Agent description
		initConfig,                  // Callback for initializing the agent config
		run,                         // Callback for executing the agent
		corecfg.TraceabilityAgent,   // Agent Type (Discovery or Traceability)
	)

	/ Get the root command properties and bind the config property in YAML definition
	rootProps := RootCmd.GetProperties()
	rootProps.AddStringProperty("gateway.trafficLogFilePath", "./logs/traffic.log", "Sample log file with traffic event from gateway")
	rootProps.AddStringSliceProperty("gateway.excludeHeaders", []string{}, "List of headers to be excluded from published events")
	...
}

// Callback that agent will call to process the execution of custom elastic beat
func run() error {
	return beatCmd.Execute()
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {

	rootProps := RootCmd.GetProperties()
	// Parse the config from bound properties and setup gateway config
	gatewayConfig := &config.GatewayConfig{
		TrafficLogFilePath:  rootProps.StringPropertyValue("gateway.trafficLogFilePath"),
		ExcludeHeaders:      rootProps.StringSLICEPropertyValue("gateway.excludeHeaders"),
		...
	}

	agentConfig := &config.AgentConfig{
		CentralCfg: centralConfig,
		GatewayCfg: gatewayConfig,
	}
	return agentConfig, nil
}

```

### Initializing Agent/Custom elastic beat
The traceability agent are custom beat which needs to implement the Beater interface defined in libbeat

```
// Beater is the interface that must be implemented by every Beat. A Beater
// provides the main Run-loop and a Stop method to break the Run-loop.
// Instantiation and Configuration is normally provided by a Beat-`Creator`.
//
// Once the beat is fully configured, the Run() method is invoked. The
// Run()-method implements the beat its run-loop. Once the Run()-method returns,
// the beat shuts down.
//
// The Stop() method is invoked the first time (and only the first time) a
// shutdown signal is received. The Stop()-method normally will stop the Run()-loop,
// such that the beat can gracefully shutdown.
type Beater interface {
	// The main event loop. This method should block until signalled to stop by an
	// invocation of the Stop() method.
	Run(b *Beat) error

	// Stop is invoked to signal that the Run method should finish its execution.
	// It will be invoked at most once.
	Stop()
}
```

To implement the Beater interface define a beater object that implements the methods specified by the interface.

```
type customLogBeater struct {
	client         beat.Client
	...
}


// Run starts customLogBeater.
func (bt *customLogBeater) Run(b *beat.Beat) error {
	bt.client, err = b.Publisher.Connect()
	...
}


// Stop stops customLogTraceabilityAgent.
func (bt *customLogBeater) Stop() {
	bt.client.Close()
	...
}
```

The custom beat needs create a method to create the beater object. This method is used by the beat root command to setup (with libbeatcmd.GenRootCmdWithSettings() method) to hook the method with libbeat initialization. When the beat execution start the beat library makes a call to this method to create the beater and then make call to Run method to execute the beat processing.

```
// New creates an instance of aws_apigw_traceability_agent.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	bt := &customLogBeater{
	}

	...
	...

	return bt, nil
}
```

### Transaction Event processing and Event Generation

The goal of the traceability agents is to publish API traffic events to Amplify ingestion service to allow computing statistics and monitor the transactions. To achieve this there are two kinds of log events that can be published thru traceability agents.
- Transaction Summary : Summarizes the interaction between the client application and discovered API. The transaction summary log entry is used for computing the API Usage statistics and display the list of transactions on API Traffic page.
- Transaction Event: Represents the interaction between services involved in the API transaction and provides details on the protocol being used in the interaction. 

This means that there will be a transaction summary log entry for API transaction flow from client but there could be multiple transaction events log entries involved in a transaction flow, first (leg 0) identifying the inbound interaction from client app to API endpoint exposed by the API Gateway and second(leg 1) identifying outbound interaction from API Gateway to backend API.

#### Sample Transaction Summary
```
{
    "version": "1.0",
    "timestamp": 1607622630,
    "transactionId": "0000000011",
    "environment": "sample-demo",
    "apicDeployment": "preprod",
    "environmentId": "8a0597c6762adbac01763cd9445800a0",
    "tenantId": "224879455212557",
    "trcbltPartitionId": "224879455212557",
    "type": "transactionSummary",
    "transactionSummary": {
        "status": "Success",
        "statusDetail": "200",
        "team": {
            "id": "e4eb11a969e09fdf0169e3d994e2000a"
        },
        "proxy": {
            "id": "unknown"
        },
        "entryPoint": {
            "type": "http",
            "method": "GET",
            "path": "/api/v1/test",
            "host": "localhost"
        }
    }
}
```

#### Sample Transaction Event
```
{
    "version": "1.0",
    "timestamp": 1607622630,
    "transactionId": "0000000011",
    "environment": "sample-demo",
    "apicDeployment": "preprod",
    "environmentId": "8a0597c6762adbac01763cd9445800a0",
    "tenantId": "224879455212557",
    "trcbltPartitionId": "224879455212557",
    "type": "transactionEvent",
    "transactionEvent": {
        "id": "0000000001.2",
        "parentId": "0000000001.1",
        "source": "localhost:9090",
        "destination": "somehost:443",
        "direction": "OUTBOUND",
        "status": "PASS",
        "protocol": {
            "type": "http",
            "uri": "/api/v1/test",
            "method": "GET",
            "status": 200,
            "statusText": "OK",
            "host": "localhost",
            "remoteAddr": "somehost",
            "remotePort": 443,
            "localAddr": "localhost",
            "localPort": 9090,
            "requestHeaders": "{\"X-Header-1\":\"value\"}",
            "responseHeaders": "{\"Content-Length\":\"1000\",\"Content-Type\":\"application/json\"}"
        }
    }
}
```

#### Common Log Entry attributes
The attributes below are to be included in both type of events.

| Attribute Name    | Description                                                                                                      |
|-------------------|------------------------------------------------------------------------------------------------------------------|
| version           | Event version (should be set to "1.0")                                                                           |
| timestamp         | Unix timestamp of the event                                                                                      |
| transactionId     | Unique business transaction id. Used for correlate the events for same transaction                               |
| environmentName   | Name of the Amplify Central environment                                                                          |
| environmentId     | ID of the Amplify Central environment                                                                            |
| apicDeployment    | Name of APIC deployment environment (prod, prod-eu)                                                              |
| tenantId          | Amplify platform organization identifier                                                                         |
| trcbltPartitionId | Amplify platform organization identifier. Used by Amplify Ingestion service to send events to appropriate tenant |
| type              | Identifies the type of log event (transactionSummary or transactionEvent)                                        |


#### Transaction Summary attributes

| Attribute Name    | Description                                                                                                    |
|-------------------|----------------------------------------------------------------------------------------------------------------|
| status            | Identifies the transaction flow status (Success, Failure, Exception or Unknown )                               |
| statusDetail      | Protocol specific response status                                                                              |
| duration          | Duration in milliseconds                                                                                       |
| proxy.id          | ID of the API on remote gateway. This should be prefixed with "remoteApiId_". For e.g. "remoteApiId_0000000001 |
| proxy.name        | Name of the API on remote gateway                                                                              |
| proxy.revision    | Revision of the API on remote gateway                                                                          |
| team.id           | Amplify Team ID                                                                                                |
| team.name         | Amplify Team Name                                                                                              |
| entryPoint.type   | Protocol type used in transaction flow (http)                                                                  |
| entryPoint.method | HTTP Method                                                                                                    |
| entryPoint.path   | HTTP Path                                                                                                      |
| entryPoint.host   | HTTP Host request header                                                                                       |

#### Transaction Event attributes

| Attribute Name | Description                                                                                          |
|----------------|------------------------------------------------------------------------------------------------------|
| id             | Id of the transaction                                                                                |
| parentId       | Id of parent transaction                                                                             |
| source         | Name of source service                                                                               |
| destination    | Name of destination service                                                                          |
| duration       | Duration in milliseconds                                                                             |
| direction      | Direction of the transaction(Inbound, Outbound). Inbound to API Gateway or outbound from API Gateway |
| status         | The status of the transaction leg                                                                    |
| protocol       | Protocol(http or jms) specific details                                                               |

##### HTTP Protocol specific attributes

| Attribute Name  | Description                                |
|-----------------|--------------------------------------------|
| type            | Identifies the HTTP protocol ("http")      |
| uri             | HTTP URI                                   |
| method          | HTTP Method                                |
| status          | HTTP Status Code                           |
| statusText      | String representation of HTTP Status Code  |
| userAgent       | User Agent used in the transaction         |
| host            | HTTP Host request header                   |
| version         | HTTP Version                               |
| bytesReceived   | Total bytes received                       |
| bytesSent       | Total bytes sent                           |
| remoteName      | Remote host name                           |
| remoteAddr      | Remote host address                        |
| remotePort      | Remote port                                |
| localAddr       | Local address                              |
| localPort       | Local port                                 |
| requestHeaders  | Request headers in serialized json format  |
| responseHeaders | Response headers in serialized json format |

The Agent SDK provides the structures with above definition to setup the log entries for both type of events
```
type LogEvent struct {
	Version            string   `json:"version"`
	Stamp              int64    `json:"timestamp"`
	TransactionID      string   `json:"transactionId"`
	Environment        string   `json:"environment,omitempty"`
	APICDeployment     string   `json:"apicDeployment,omitempty"`
	EnvironmentName    string   `json:"environmentName"`
	EnvironmentID      string   `json:"environmentId"`
	TenantID           string   `json:"tenantId"`
	TrcbltPartitionID  string   `json:"trcbltPartitionId"`
	Type               string   `json:"type"`
	TargetPath         string   `json:"targetPath,omitempty"`
	ResourcePath       string   `json:"resourcePath,omitempty"`
	TransactionEvent   *Event   `json:"transactionEvent,omitempty"`
	TransactionSummary *Summary `json:"transactionSummary,omitempty"`
}
```

The traffic log entry from different API gateways can be in different formats, so the agents can parse the received log entry and map the entries to TransactionSummary or TransactionEvent log events. The Agent SDK provides the following set of builders to setup the both type of log events.

```
// SummaryBuilder - Interface to build the log event of type transaction summary
type SummaryBuilder interface {
	SetTimestamp(timestamp int64) SummaryBuilder
	SetTransactionID(transactionID string) SummaryBuilder
	SetAPICDeployment(apicDeployment string) SummaryBuilder
	SetEnvironmentName(environmentName string) SummaryBuilder
	SetEnvironmentID(environmentID string) SummaryBuilder
	SetTenantID(tenantID string) SummaryBuilder
	SetTrcbltPartitionID(trcbltPartitionID string) SummaryBuilder
	SetTargetPath(targetPath string) SummaryBuilder
	SetResourcePath(resourcePath string) SummaryBuilder

	SetStatus(status TxSummaryStatus, statusDetail string) SummaryBuilder
	SetDuration(duration int) SummaryBuilder
	SetApplication(appID, appName string) SummaryBuilder
	SetProduct(product string) SummaryBuilder
	SetTeam(teamID string) SummaryBuilder
	SetProxy(proxyID, proxyName string, proxyRevision int) SummaryBuilder
	SetRunTime(runtimeID, runtimeName string) SummaryBuilder
	SetEntryPoint(entryPointType, method, path, host string) SummaryBuilder

	Build() (*LogEvent, error)
}

// EventBuilder - Interface to build the log event of type transaction event
type EventBuilder interface {
	SetTimestamp(timestamp int64) EventBuilder
	SetTransactionID(transactionID string) EventBuilder
	SetAPICDeployment(apicDeployment string) EventBuilder
	SetEnvironmentName(environmentName string) EventBuilder
	SetEnvironmentID(environmentID string) EventBuilder
	SetTenantID(tenantID string) EventBuilder
	SetTrcbltPartitionID(trcbltPartitionID string) EventBuilder
	SetTargetPath(targetPath string) EventBuilder
	SetResourcePath(resourcePath string) EventBuilder

	SetID(id string) EventBuilder
	SetParentID(parentID string) EventBuilder
	SetSource(source string) EventBuilder
	SetDestination(destination string) EventBuilder
	SetDuration(duration int) EventBuilder
	SetDirection(direction string) EventBuilder
	SetStatus(status TxEventStatus) EventBuilder
	SetProtocolDetail(protocolDetail interface{}) EventBuilder

	Build() (*LogEvent, error)
}

// HTTPProtocolBuilder - Interface to build the HTTP protocol details for transaction log event
type HTTPProtocolBuilder interface {
	SetURI(uri string) HTTPProtocolBuilder
	SetVersion(version string) HTTPProtocolBuilder
	SetArgs(args string) HTTPProtocolBuilder
	SetArgsMap(args map[string][]string) HTTPProtocolBuilder
	AddArg(key string, value []string) HTTPProtocolBuilder
	SetMethod(method string) HTTPProtocolBuilder
	SetStatus(status int, statusText string) HTTPProtocolBuilder
	SetUserAgent(userAgent string) HTTPProtocolBuilder
	SetHost(host string) HTTPProtocolBuilder
	SetByteLength(byteReceived, byteSent int) HTTPProtocolBuilder
	SetRemoteAddress(remoteName string, remoteAddr string, remotePort int) HTTPProtocolBuilder
	SetLocalAddress(localAddr string, localPort int) HTTPProtocolBuilder
	SetSSLProperties(sslProtocol, sslServerName, sslSubject string) HTTPProtocolBuilder
	SetAuthSubjectID(authSubjectID string) HTTPProtocolBuilder
	SetHeaders(requestHeaders, responseHeaders string) HTTPProtocolBuilder
	SetRequestHeaders(requestHeaders map[string]string) HTTPProtocolBuilder
	SetResponseHeaders(responseHeaders map[string]string) HTTPProtocolBuilder
	AddRequestHeader(headerKey string, headerValue string) HTTPProtocolBuilder
	AddResponseHeader(headerKey string, headerValue string) HTTPProtocolBuilder
	SetIndexedHeaders(indexedRequestHeaders, indexedResponseHeaders string) HTTPProtocolBuilder
	SetIndexedRequestHeaders(indexedRequestHeaders map[string]string) HTTPProtocolBuilder
	SetIndexedResponseHeaders(indexedResponseHeaders map[string]string) HTTPProtocolBuilder
	AddIndexedRequestHeader(headerKey string, headerValue string) HTTPProtocolBuilder
	AddIndexedResponseHeader(headerKey string, headerValue string) HTTPProtocolBuilder
	SetPayload(requestPayload, responsePayload string) HTTPProtocolBuilder
	SetWAFStatus(wasStatus int) HTTPProtocolBuilder

	Build() (TransportProtocol, error)
}

// JMSProtocolBuilder - Interface to build the JMS protocol details for transaction log event
type JMSProtocolBuilder interface {
	SetMessageID(messageID string) JMSProtocolBuilder
	SetCorrelationID(correlationID string) JMSProtocolBuilder
	SetAuthSubjectID(authSubjectID string) JMSProtocolBuilder
	SetDestination(destination string) JMSProtocolBuilder
	SetProviderURL(providerURL string) JMSProtocolBuilder
	SetDeliveryMode(deliveryMode int) JMSProtocolBuilder
	SetPriority(priority int) JMSProtocolBuilder
	SetReplyTo(replyTo string) JMSProtocolBuilder
	SetRedelivered(redelivered int) JMSProtocolBuilder
	SetTimestamp(timestamp int) JMSProtocolBuilder
	SetExpiration(expiration int) JMSProtocolBuilder
	SetJMSType(jmsType string) JMSProtocolBuilder
	SetStatus(status string) JMSProtocolBuilder
	SetStatusText(statusText string) JMSProtocolBuilder

	Build() (TransportProtocol, error)
}
```

The sample code below demonstrates building up the transaction summary log event
```
 	txSummary, err := transaction.NewTransactionSummaryBuilder().
		SetTimestamp(eventTime).
		SetTransactionID(txID).
		SetStatus(m.getTransactionSummaryStatus(statusCode), strconv.Itoa(statusCode)).
		SetTeam(teamID).
		SetEntryPoint("http", method, uri, host).
		SetProxy("unknown", "", 0).
		Build()
```

Below is an example code for building transaction event with HTTP protocol details
```
	httpProtocolDetails, err := transaction.NewHTTPProtocolBuilder().
		SetURI(txDetails.URI).
		SetMethod(txDetails.Method).
		SetStatus(txDetails.StatusCode, http.StatusText(txDetails.StatusCode)).
		SetHost(txDetails.SourceHost).
		SetHeaders(m.buildHeaders(txDetails.RequestHeaders), m.buildHeaders(txDetails.ResponseHeaders)).
		SetByteLength(txDetails.RequestBytes, txDetails.ResponseBytes).
		SetRemoteAddress("", txDetails.DesHost, txDetails.DestPort).
		SetLocalAddress(txDetails.SourceHost, txDetails.SourcePort).
		Build()
	...
	...
	txEvent, err := transaction.NewTransactionEventBuilder().
		SetTimestamp(eventTime).
		SetTransactionID(txID).
		SetID(eventID).
		SetParentID(parentEventID).
		SetSource(txDetails.SourceHost + ":" + strconv.Itoa(txDetails.SourcePort)).
		SetDestination(txDetails.DesHost + ":" + strconv.Itoa(txDetails.DestPort)).
		SetDirection(direction).
		SetStatus(m.getTransactionEventStatus(txDetails.StatusCode)).
		SetProtocolDetail(httpProtocolDetails).
		Build()
```

The mapped LogEvent object can then be used to construct beat.Event using the transaction.EventGenerator. The agents can construct event generator using transaction.NewEventGenerator() method.

Below is the sample code for the custom beat generating the events. The sample does not demonstrate how the agent collects the log entry for API Gateway and is left up to agent implementation.

```
func (bt *customLogBeater) Run(b *beat.Beat) error {
	...
	bt.client, err = b.Publisher.Connect()

	// Construct the event generator
	eventGenerator = transaction.NewEventGenerator(),
	...

	for {
		select {
		...
		// Receive the collected log entry
		case eventData := <-bt.eventChannel:
			// Parse the gateway log entry
			var gatewayTrafficLogEntry GwTrafficLogEntry
			json.Unmarshal(rawEventData, &gatewayTrafficLogEntry)

			// Map the gateway log entry to transaction.LogEvents for TransactionSummary and TransactionEvent
			logEvents := p.eventMapper.processMapping(gatewayTrafficLogEntry)

			// Use event generator to create beat.Events for logEvents 
			eventsToPublish := make([]beat.Event, 0)
			for _, logEvent := range logEvents {
				// Generates the beat.Event with attributes by Amplify ingestion service
				event, _ := eventGenerator.CreateEvent(logEvent, time.Now(), nil, nil, nil)
				events = append(events, event)
			}

			// Publish the events to transport
			bt.client.PublishAll(eventsToPublish)
		}
	}
}
```

The above sample demonstrates the event generation in the component that collects data, however the developer might want to use existing beat implementation(like filebeat) which has its own data collection mechanism that publishes event to the component processing the output. The Agent SDK provides mechanism to hook a callback that can be invoked before the event is published over the transport.
To use the output event process the agent needs to implement the Beater interface defined in libbeat

```
type OutputEventProcessor interface {
	Process(events []publisher.Event) []publisher.Event
}
```

Below is the sample code demonstrating the use of output event processor.

```
import (
		filebeater "github.com/elastic/beats/v7/filebeat/beater"
		...
)

// Custom beat factory method to create wrapped filebeat
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	...

	traceability.SetOutputEventProcessor(eventProcessor)
	...

	return filebeater.New(b, cfg)
}

type EventProcessor struct {
	...
}

// Process - callback set as output event processor that gets invoked by transport publisher to process the received events
func (p *EventProcessor) Process(events []publisher.Event) []publisher.Event {
	newPublisherEvents := make([]publisher.Event, 0)
	for _, event := range events {
		// Parse the log entry received from API Gateway logs
		rawEventData, err := event.Content.Fields.GetValue("message")
		...
		var gatewayTrafficLogEntry GwTrafficLogEntry
		json.Unmarshal(rawEventData, &gatewayTrafficLogEntry)

		// Map the gateway log entry to transaction.LogEvents for TransactionSummary and TransactionEvent
		logEvents := p.eventMapper.processMapping(gatewayTrafficLogEntry)

		// Use event generator to create beat.Events for logEvents 
		for _, logEvent := range logEvents {
			// Generates the beat.Event with attributes by Amplify ingestion service
			beatEvent, _ := eventGenerator.CreateEvent(logEvent, time.Now(), nil, nil, nil)
			publisherEvent := publisher.Event{
				Content: beatEvent,
			}
			newPublisherEvents = append(newEvents, publisherEvent)
		}
	}
	return newPublisherEvents
}

```

### Traceability redaction

The agent SDK has the ability to handle redaction and sanitization of URL path, Query Arguments, Request and Response headers.  When building the transaction summary and protocol events the SDK will apply these rules before sending to Amplify Central.

By default all of the fields URL path, Query Arguments, Request and Response headers will be redacted and sanitized.  The settings below are to enable what to allow through.

Below is the list of the redaction configuration properties in a YAML and their corresponding environment variables that can be set to override the config in YAML.  All of these are children of output.traceability.redaction

| YAML property           | Variable name                                  | Description                                                                  |
|-------------------------|------------------------------------------------|------------------------------------------------------------------------------|
| path.show               | TRACEABILITY_REDACTION_PATH_SHOW               | Determines what path values to send to Amplify                               |
| queryArgument.show      | TRACEABILITY_REDACTION_QUERYARGUMENT_SHOW      | Determines what query arguments to send to Amplify                           |
| queryArgument.sanitize  | TRACEABILITY_REDACTION_QUERYARGUMENT_SANITIZE  | Determines what query argument values to sanitize before sending to Amplify  |
| requestHeader.show      | TRACEABILITY_REDACTION_REQUESTHEADER_SHOW      | Determines what request headers to send to Amplify                           |
| requestHeader.sanitize  | TRACEABILITY_REDACTION_REQUESTHEADER_SANITIZE  | Determines what request header values to sanitize before sending to Amplify  |
| responseHeader.show     | TRACEABILITY_REDACTION_QUERYARGUMENT_SHOW      | Determines what response headers to send to Amplify                          |
| responseHeader.sanitize | TRACEABILITY_REDACTION_RESPONSEHEADER_SANITIZE | Determines what response header values to sanitize before sending to Amplify |

#### Setting up redaction and sanitization

All of the SHOW properties above accept a keyMatch attribute that is a regular expressions to determine what path value, argument, or header may be sent to Amplify.

All of the SANITIZE properties above accept both a keyMatch and valueMatch attribute.  The keyMatch attribute is applied to the argument name or header key. Once it finds a match the valueMatch regular expression is applied to the value to remove any string of characters that match.

In order for the Sanitization to take place the argument name or header key must first have passed a show keyMatch.

The regular expression syntax that may be applied can be found [here](https://golang.org/pkg/regexp/syntax/)

#### Setting up redaction in YAML

```yaml
output.traceability:
...
  redaction:
    path:
      show:
        keyMatch: "^path" # allow paths values that start with the word path
    queryArgument:
      show:
        keyMatch: "^id$" # allow query arguments that equal id exactly
      sanitize:
        keyMatch: "^id$" # find the query arguments that equal id exactly
        valueMatch: "^.*{0,5}" # sanitize the first five characters, no matter what they are
    requestHeader:
      show:
        keyMatch: "header" # allow request headers that have the word header within them
      sanitize:
        keyMatch: "header" # find request headers that have the word header within them
        valueMatch: ".*" # sanitize the entire header value
    responseHeader:
      show:
        keyMatch: ".*" # allow all response headers
      sanitize:
        keyMatch: "^response" # find any response headers that start with the word response
        valueMatch: "password" # sanitize the word password any time it occurs in the header value 
```

#### Using environment variables for redaction

The default YAML in the example has the following setup for the redaction section. As long as this is not changed the environment variables below can set up the redaction.

```yaml
output.traceability:
...
  redaction:
    path:
      show: ${TRACEABILITY_REDACTION_PATH_SHOW:[]}
    queryArgument:
      show: ${TRACEABILITY_REDACTION_QUERYARGUMENT_SHOW:[]}
      sanitize: ${TRACEABILITY_REDACTION_QUERYARGUMENT_SANITIZE:[]}
    requestHeader:
      show: ${TRACEABILITY_REDACTION_REQUESTHEADER_SHOW:[]}
      sanitize: ${TRACEABILITY_REDACTION_REQUESTHEADER_SANITIZE:[]}
    responseHeader:
      show: ${TRACEABILITY_REDACTION_RESPONSEHEADER_SHOW:[]}
      sanitize: ${TRACEABILITY_REDACTION_RESPONSEHEADER_SANITIZE:[]}
```

When setting up the environment variables the following is the syntax that must be used.  These settings are the same as in the YAML [example](#setting-up-redaction-in-yaml)

```ini
TRACEABILITY_REDACTION_PATH_SHOW=[{keyMatch:"^path"}]
TRACEABILITY_REDACTION_QUERYARGUMENT_SHOW=[{keyMatch:"^id$"}]
TRACEABILITY_REDACTION_QUERYARGUMENT_SANITIZE=[{keyMatch:"^id$",valueMatch:"^.*{0,5}"}]
TRACEABILITY_REDACTION_REQUESTHEADER_SHOW=[{keyMatch:"header"}]
TRACEABILITY_REDACTION_REQUESTHEADER_SANITIZE=[{keyMatch:"header",valueMatch:".*"}]
TRACEABILITY_REDACTION_RESPONSEHEADER_SHOW=[{keyMatch:".*"}]
TRACEABILITY_REDACTION_RESPONSEHEADER_SANITIZE=[{keyMatch:"^response",valueMatch:"password"}]
```

### Traceability sampling

The agent SDK has the ability to handle sampling of transactions that are processed.  This sampling controls what transaction events are sent to Amplify.

By default all transaction data is sent to Amplify.

Below is the list of the sampling configuration properties in a YAML and their corresponding environment variables that can be set to override the config in YAML.  All of these are children of output.traceability.sampling

| YAML property | Variable name                    | Description                                                       |
|---------------|----------------------------------|-------------------------------------------------------------------|
| percentage    | TRACEABILITY_SAMPLING_PERCENTAGE | Defines the percentage of events (0-100) that are sent to Amplify |

### Building the Agent

The agents are applications built using [Go programming language](https://golang.org/). Go is open source programming language that gets statically compiled and comes with a rich toolset to obtain packages and building executables. The Agents SDK uses the Go module as the dependency management which was introduced in Go 1.11. Go modules is collection of packages with go.mod file in its root directory which defines the modules source paths used in the packages as imports.

The *go mod tidy* command will prune any unused dependencies from your *go.mod* and update the files to include used dependencies. The *go mod verify* command checks the dependencies, downloads them from the source repository and updates the cryptographic hashes in your go.sum file. 

Run the following commands to resolve the dependencies
```
go mod tidy
go mod verify
```

To build the agent once the dependencies are resolved *go build* command can be used which compile the source and generates the binary executable for the target system. 
The Agent SDK provides support for specifying the version of the agent at the build time. The following variables can be set by compile flags to setup agent name, version, commit SHA and build time.

- github.com/Axway/agent-sdk/pkg/cmd.BuildTime
- github.com/Axway/agent-sdk/pkg/cmd.BuildVersion
- github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha
- github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName

The following is an example of the build command that can be configured in the Makefile

```
@export time=`date +%Y%m%d%H%M%S` && \
export version=`cat version` && \
export commit_id=`git rev-parse --short HEAD` && \
go build -tags static_all \
	-ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=$${time}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=$${version}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=$${commit_id}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=SampleTraceabilityAgent'" \
	-a -o ${WORKSPACE}/bin/apic_traceability_agent ${WORKSPACE}/main.go
```

#### Pre-requisites for executing the agent
* An Axway Amplify Central subscription in the Amplify™ platform. See [Get started with Amplify Central](https://axway-open-docs.netlify.app/docs/central/quickstart).
* An Amplify Central Service Account. See [Create a service account](https://axway-open-docs.netlify.app/docs/central/cli_central/cli_install/#22-create-a-service-account-using-the-Amplify-central-ui).
* An Amplify Central environment. See [Create environment](https://axway-open-docs.netlify.app/docs/central/cli_central/cli_environments/#create-an-environment).

### Executing Traceability Agent
The Agent built using Amplify Central Agents SDK can be executed by running the executable. The agent on initialization tries to load the configuration from following sources and applies the configuration properties in the order described below.

1. The configuration YAML file in the current working directory.
2. Environment variable defined for configuration override on the shell executing the agent
3. A file containing the list of environment variable override
4. Command line flags

Below is the sample for executing the agent when the config YAML file only is used.
```
cd <path-to-agent-install-directory>
./apic_traceability_agent
```

Typically, the configuration YAML can be placed in the same directory as the agent executable, but alternatively the YAML file could be placed in another directory and then *pathConfig* command line flags can be used to specify the directory path containing the YAML file.
```
<path-to-agent-install-directory>/apic_traceability_agent --pathConfig <directory-path-for-agent-yaml-config-file>
```

The following is an example of command to execute the agent with a file holding environment variables 
```
cd <path-to-agent-install-directory>
./apic_traceability_agent --envFile <path-of-env-file>/config.env
```

The agent configuration can also be passed as command line flags. Below is an example of agent usage that details the command line flags and configuration properties
```
cd <path-to-agent-install-directory>
./apic_traceability_agent --help
Sample Traceability Agent

Usage:
  apic_traceability_agent [flags]
  apic_traceability_agent [command]

Available Commands:
  export      Export current config or index template
  help        Help about any command
  keystore    Manage secrets keystore
  run         Run apic_traceability_agent
  setup       Setup index template, dashboards and ML jobs
  test        Test config
  version     Show current version info

Flags:
  -E, --E setting=value                      Configuration overwrite
  -N, --N                                    Disable actual publishing for testing
  -c, --c string                             Configuration file, relative to path.config (default "apic_traceability_agent.yml")
      --centralAgentName string              The name of the asociated agent resource in Amplify Central
      --centralApiServerVersion string       Version of the API Server (default "v1alpha1")
      --centralAuthClientId string           Client ID for the service account
      --centralAuthKeyPassword string        Password for the private key, if needed
      --centralAuthPrivateKey string         Path to the private key for Amplify Central Authentication (default "/etc/private_key.pem")
      --centralAuthPublicKey string          Path to the public key for Amplify Central Authentication (default "/etc/public_key")
      --centralAuthRealm string              Amplify Central authentication Realm (default "Broker")
      --centralAuthTimeout duration          Timeout waiting for AxwayID response (default 10s)
      --centralAuthUrl string                Amplify Central authentication URL (default "https://login.axway.com/auth")
      --centralDeployment string             Amplify Central (default "prod")
      --centralEnvironment string            The Environment that the APIs will be associated with in Amplify Central
      --centralOrganizationID string         Tenant ID for the owner of the environment
      --centralPlatformURL string            URL of the platform (default "https://platform.axway.com")
      --centralPollInterval duration         The time interval at which the central will be polled for subscription processing. (default 1m0s)
      --centralProxyUrl string               The Proxy URL to use for communication to Amplify Central
      --centralSslCipherSuites strings       List of supported cipher suites, comma separated (default [ECDHE-ECDSA-AES-256-GCM-SHA384,ECDHE-RSA-AES-256-GCM-SHA384,ECDHE-ECDSA-CHACHA20-POLY1305,ECDHE-RSA-CHACHA20-POLY1305,ECDHE-ECDSA-AES-128-GCM-SHA256,ECDHE-RSA-AES-128-GCM-SHA256,ECDHE-ECDSA-AES-128-CBC-SHA256,ECDHE-RSA-AES-128-CBC-SHA256])
      --centralSslInsecureSkipVerify         Controls whether a client verifies the server's certificate chain and host name
      --centralSslMaxVersion string          Maximum acceptable SSL/TLS protocol version (default "0")
      --centralSslMinVersion string          Minimum acceptable SSL/TLS protocol version (default "TLS1.2")
      --centralSslNextProtos strings         List of supported application level protocols, comma separated
      --centralTeam string                   Team name for creating catalog
      --centralUrl string                    URL of Amplify Central (default "https://apicentral.axway.com")
      --cpuprofile string                    Write cpu profile to file
  -d, --d string                             Enable certain debug selectors
  -e, --e                                    Log to stderr and disable syslog/file output
      --envFile string                       Path of the file with environment variables to override configuration
      --environment environmentVar           set environment the Beat is run in (default default)
      --gateway-sectionConfig_key_1 string   Sample Config Key 1
      --gateway-sectionConfig_key_2 string   Sample Config Key 1
      --gateway-sectionConfig_key_3 string   Sample Config Key 3
      --gateway-sectionLogFile string        Sample log file with traffic event from gateway (default "./logs/traffic.log")
      --gateway-sectionProcessOnInput        Flag to process received event on input or by output before publishing the event by transport (default true)
  -h, --help                                 help for apic_traceability_agent
      --httpprof string                      Start pprof http server
      --logFileCleanbackups int              The maximum number of days, 24 hour periods, to keep the log file backps
      --logFileKeepfiles int                 The maximum number of backups to keep of log files (default: 7) (default 7)
      --logFileName string                   Name of the log files (default "apic_traceability_agent.log")
      --logFilePath string                   Log file path if output type is file or both (default "logs")
      --logFileRotateeverymegabytes int      The maximum size of a log file, in megabytes  (default: 100) (default 100)
      --logFormat string                     Log format (json, line) (default "json")
      --logLevel string                      Log level (debug, info, warn, error) (default "info")
      --logMaskedValues string               List of key words in the config to be masked (e.g. pwd, password, secret, key
      --logOutput string                     Log output type (stdout, file, both) (default "stdout")
      --memprofile string                    Write memory profile to this file
      --path.config string                   Configuration path
      --path.data string                     Data path
      --path.home string                     Home path
      --path.logs string                     Logs path
      --pathConfig string                    Configuration file path for the agent
      --plugin pluginList                    Load additional plugins
      --status                               Get the status of all the Health Checks
      --statusHealthCheckInterval duration   Time between running periodic health checker. Can be between 30 seconds and 5 minutes (binary agents only) (default 30s)
      --statusHealthCheckPeriod duration     Time in minutes allotted for services to be ready before exiting discovery agent (default 3m0s)
      --statusPort int                       The port that will serve the status endpoints (default 8989)
      --strict.perms                         Strict permission checking on config files (default true)
      --synchronize                          Run the sync process for the discovery agent
  -v, --v                                    Log at INFO level
      --version                              version for apic_traceability_agent

```
