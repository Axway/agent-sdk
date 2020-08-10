package apic

import (
	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
)

// Various consts for use
const (
	API           = "API"
	Wsdl          = "wsdl"
	SwaggerV2     = "swaggerv2"
	Oas2          = "oas2"
	Oas3          = "oas3"
	Specification = "specification"
	Swagger       = "swagger"

	SubscriptionSchemaNameSuffix      = ".authsubscription"
	DefaultSubscriptionWebhookName    = "subscriptionwebhook"
	DefaultSubscriptionWebhookAuthKey = "webhookAuthKey"
)

type apiErrorResponse map[string][]apiError

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// consts for state
const (
	UnpublishedState = "UNPUBLISHED"
	PublishedState   = "PUBLISHED"
)

// consts for update serverity
const (
	MajorChange = "MAJOR"
	MinorChange = "MINOR"
)

//ServiceBody -
type ServiceBody struct {
	NameToPush        string `json:",omitempty"`
	APIName           string `json:",omitempty"`
	RestAPIID         string `json:",omitempty"`
	URL               string `json:",omitempty"`
	Stage             string `json:",omitempty"`
	TeamID            string `json:",omitempty"`
	Description       string `json:",omitempty"`
	Version           string `json:",omitempty"`
	AuthPolicy        string `json:",omitempty"`
	Swagger           []byte `json:",omitempty"`
	Documentation     []byte `json:",omitempty"`
	Tags              map[string]interface{}
	Buffer            []byte            `json:",omitempty"`
	AgentMode         corecfg.AgentMode `json:",omitempty"`
	ServiceExecution  serviceExecution  `json:"omitempty"`
	Image             string
	ImageContentType  string
	CreatedBy         string
	ResourceType      string
	SubscriptionName  string
	APIUpdateSeverity string `json:",omitempty"`
}

// ServiceClient -
type ServiceClient struct {
	tokenRequester                     tokenGetter
	cfg                                corecfg.CentralConfig
	apiClient                          coreapi.Client
	DefaultSubscriptionSchema          SubscriptionSchema
	RegisteredSubscriptionSchema       SubscriptionSchema
	subscriptionMgr                    SubscriptionManager
	DefaultSubscriptionApprovalWebhook corecfg.WebhookConfig
}

// APIServerInfoProperty -
type APIServerInfoProperty struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// APIServerInfo -
type APIServerInfo struct {
	ConsumerInstance APIServerInfoProperty `json:"consumerInstance,omitempty"`
	Environment      APIServerInfoProperty `json:"environment,omitempty"`
}

// APIServerScope -
type APIServerScope struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

// APIServerReference -
type APIServerReference struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// APIServerMetadata -
type APIServerMetadata struct {
	ID         string               `json:"id,omitempty"`
	Scope      *APIServerScope      `json:"scope,omitempty"`
	References []APIServerReference `json:"references,omitempty"`
}

// APIServer -
type APIServer struct {
	Name       string                 `json:"name"`
	Title      string                 `json:"title"`
	Tags       []string               `json:"tags"`
	Attributes map[string]interface{} `json:"attributes"`
	Spec       interface{}            `json:"spec"`
	Metadata   *APIServerMetadata     `json:"metadata,omitempty"`
}

// APIServiceSpec -
type APIServiceSpec struct {
	Description string          `json:"description"`
	Icon        *APIServiceIcon `json:"icon,omitempty"`
}

// APIServiceRevisionSpec -
type APIServiceRevisionSpec struct {
	APIService string             `json:"apiService"`
	Definition RevisionDefinition `json:"definition"`
}

// RevisionDefinition -
type RevisionDefinition struct {
	Type  string `json:"type,omitempty"`
	Value []byte `json:"value,omitempty"`
}

// APIServiceIcon -
type APIServiceIcon struct {
	ContentType string `json:"contentType"`
	Data        string `json:"data"`
}

// APIServerInstanceSpec -
type APIServerInstanceSpec struct {
	APIServiceRevision string     `json:"apiServiceRevision,omitempty"`
	InstanceEndPoint   []EndPoint `json:"endpoint,omitempty"`
}

// EndPoint -
type EndPoint struct {
	Host     string   `json:"host,omitempty"`
	Port     int      `json:"port,omitempty"`
	Protocol string   `json:"protocol,omitempty"`
	Routing  BasePath `json:"routing,omitempty"`
}

// BasePath -
type BasePath struct {
	Path string `json:"basePath,omitempty"`
}

//EnvironmentSpec - structure of environment returned when not using API Server
type EnvironmentSpec struct {
	ID       string      `json:"id,omitempty"`
	Name     string      `json:"name,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}
