package apic

import (
	"fmt"
	"regexp"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	maxDescriptionLength = 350
	strEllipsis          = "..."
)

var pathRe *regexp.Regexp

func init() {
	pathRe, _ = regexp.Compile("^/")
}

// ServiceBuilder - Interface to build the service body
type ServiceBuilder interface {
	SetID(ID string) ServiceBuilder
	SetPrimaryKey(key string) ServiceBuilder
	SetRequestDefinitionsAllowed(previouslyPublished bool) ServiceBuilder
	SetTitle(title string) ServiceBuilder
	SetAPIName(apiName string) ServiceBuilder
	SetURL(url string) ServiceBuilder
	SetStage(stage string) ServiceBuilder
	SetStageDescriptor(stageDescriptor string) ServiceBuilder
	SetStageDisplayName(stageName string) ServiceBuilder
	SetDescription(description string) ServiceBuilder
	SetVersion(version string) ServiceBuilder
	SetAuthPolicy(authPolicy string) ServiceBuilder
	SetAPISpec(spec []byte) ServiceBuilder
	SetDocumentation(documentation []byte) ServiceBuilder
	SetTags(tags map[string]interface{}) ServiceBuilder
	SetImage(image string) ServiceBuilder
	SetImageContentType(imageContentType string) ServiceBuilder
	SetResourceType(resourceType string) ServiceBuilder
	SetSubscriptionName(subscriptionName string) ServiceBuilder
	SetAPIUpdateSeverity(apiUpdateSeverity string) ServiceBuilder
	SetState(state string) ServiceBuilder
	SetStatus(status string) ServiceBuilder
	SetServiceAttribute(serviceAttribute map[string]string) ServiceBuilder
	SetInstanceAttribute(instanceAttribute map[string]string) ServiceBuilder
	SetRevisionAttribute(revisionAttribute map[string]string) ServiceBuilder
	SetServiceEndpoints(endpoints []EndpointDefinition) ServiceBuilder
	AddServiceEndpoint(protocol, host string, port int32, basePath string) ServiceBuilder
	SetCredentialRequestDefinitions(credentialRequestDefNames []string) ServiceBuilder
	AddCredentialRequestDefinition(credentialRequestDefName string) ServiceBuilder
	SetAccessRequestDefinitionName(accessRequestDefName string, isUnique bool) ServiceBuilder
	SetIgnoreSpecBasedCreds(ignore bool) ServiceBuilder

	SetUnstructuredType(assetType string) ServiceBuilder
	SetUnstructuredContentType(contentType string) ServiceBuilder
	SetUnstructuredLabel(label string) ServiceBuilder
	SetUnstructuredFilename(filename string) ServiceBuilder
	SetTeamName(teamName string) ServiceBuilder
	SetServiceAgentDetails(attr map[string]interface{}) ServiceBuilder
	SetInstanceAgentDetails(attr map[string]interface{}) ServiceBuilder
	SetRevisionAgentDetails(attr map[string]interface{}) ServiceBuilder

	SetSourceDataplaneType(dataplaneType DataplaneType, isDesign bool) ServiceBuilder
	SetReferenceServiceName(serviceName, envName string) ServiceBuilder
	SetReferenceInstanceName(instanceName, envName string) ServiceBuilder

	Build() (ServiceBody, error)
}

type serviceBodyBuilder struct {
	err         error
	serviceBody ServiceBody
}

// NewServiceBodyBuilder - Creates a new service body builder
func NewServiceBodyBuilder() ServiceBuilder {
	return &serviceBodyBuilder{
		serviceBody: ServiceBody{
			AuthPolicy:                Passthrough,
			authPolicies:              make([]string, 0),
			CreatedBy:                 config.AgentTypeName,
			State:                     PublishedStatus,
			Status:                    PublishedStatus,
			scopes:                    make(map[string]string),
			ServiceAttributes:         make(map[string]string),
			RevisionAttributes:        make(map[string]string),
			InstanceAttributes:        make(map[string]string),
			StageDescriptor:           "Stage",
			Endpoints:                 make([]EndpointDefinition, 0),
			UnstructuredProps:         &UnstructuredProperties{},
			credentialRequestPolicies: make([]string, 0),
			ServiceAgentDetails:       make(map[string]interface{}),
			InstanceAgentDetails:      make(map[string]interface{}),
			RevisionAgentDetails:      make(map[string]interface{}),
			requestDefinitionsAllowed: true,
			logger:                    log.NewFieldLogger().WithPackage("sdk.apic").WithComponent("serviceBuilder"),
		},
	}
}

func (b *serviceBodyBuilder) SetID(ID string) ServiceBuilder {
	b.serviceBody.RestAPIID = ID
	return b
}

func (b *serviceBodyBuilder) SetPrimaryKey(key string) ServiceBuilder {
	b.serviceBody.PrimaryKey = key
	return b
}

func (b *serviceBodyBuilder) SetRequestDefinitionsAllowed(key bool) ServiceBuilder {
	b.serviceBody.requestDefinitionsAllowed = key
	return b
}

func (b *serviceBodyBuilder) SetTitle(title string) ServiceBuilder {
	b.serviceBody.NameToPush = title
	return b
}

func (b *serviceBodyBuilder) SetAPIName(apiName string) ServiceBuilder {
	b.serviceBody.APIName = apiName
	return b
}

func (b *serviceBodyBuilder) SetURL(url string) ServiceBuilder {
	b.serviceBody.URL = url
	return b
}

func (b *serviceBodyBuilder) SetStage(stage string) ServiceBuilder {
	b.serviceBody.Stage = stage
	return b
}

func (b *serviceBodyBuilder) SetStageDescriptor(stageDescriptor string) ServiceBuilder {
	b.serviceBody.StageDescriptor = stageDescriptor
	return b
}

func (b *serviceBodyBuilder) SetStageDisplayName(stageDisplayName string) ServiceBuilder {
	b.serviceBody.StageDisplayName = stageDisplayName
	return b
}

func (b *serviceBodyBuilder) SetDescription(description string) ServiceBuilder {
	b.serviceBody.Description = description
	if len(description) > maxDescriptionLength {
		b.serviceBody.Description = description[0:maxDescriptionLength-len(strEllipsis)] + strEllipsis
	}
	return b
}

func (b *serviceBodyBuilder) SetVersion(version string) ServiceBuilder {
	b.serviceBody.Version = version
	return b
}

func (b *serviceBodyBuilder) SetAuthPolicy(authPolicy string) ServiceBuilder {
	b.serviceBody.AuthPolicy = authPolicy
	return b
}

func (b *serviceBodyBuilder) SetAPISpec(spec []byte) ServiceBuilder {
	b.serviceBody.SpecDefinition = spec
	return b
}

func (b *serviceBodyBuilder) SetDocumentation(documentation []byte) ServiceBuilder {
	b.serviceBody.Documentation = documentation
	return b
}

func (b *serviceBodyBuilder) SetTags(tags map[string]interface{}) ServiceBuilder {
	b.serviceBody.Tags = tags
	return b
}

func (b *serviceBodyBuilder) SetImage(image string) ServiceBuilder {
	b.serviceBody.Image = image
	return b
}

func (b *serviceBodyBuilder) SetImageContentType(imageContentType string) ServiceBuilder {
	b.serviceBody.ImageContentType = imageContentType
	return b
}

func (b *serviceBodyBuilder) SetResourceType(resourceType string) ServiceBuilder {
	b.serviceBody.ResourceType = resourceType
	return b
}

func (b *serviceBodyBuilder) SetResourceContentType(resourceContentType string) ServiceBuilder {
	b.serviceBody.ResourceContentType = resourceContentType
	return b
}

func (b *serviceBodyBuilder) SetSubscriptionName(subscriptionName string) ServiceBuilder {
	b.serviceBody.SubscriptionName = subscriptionName
	return b
}

func (b *serviceBodyBuilder) SetAPIUpdateSeverity(apiUpdateSeverity string) ServiceBuilder {
	b.serviceBody.APIUpdateSeverity = apiUpdateSeverity
	return b
}

func (b *serviceBodyBuilder) SetState(state string) ServiceBuilder {
	b.serviceBody.State = state
	return b
}

func (b *serviceBodyBuilder) SetStatus(status string) ServiceBuilder {
	b.serviceBody.Status = status
	return b
}

func (b *serviceBodyBuilder) SetServiceAttribute(attr map[string]string) ServiceBuilder {
	b.serviceBody.ServiceAttributes = attr
	return b
}

func (b *serviceBodyBuilder) SetInstanceAttribute(attr map[string]string) ServiceBuilder {
	b.serviceBody.InstanceAttributes = attr
	return b
}

func (b *serviceBodyBuilder) SetRevisionAttribute(attr map[string]string) ServiceBuilder {
	b.serviceBody.RevisionAttributes = attr
	return b
}

func (b *serviceBodyBuilder) SetServiceAgentDetails(attr map[string]interface{}) ServiceBuilder {
	b.serviceBody.ServiceAgentDetails = attr
	return b
}

func (b *serviceBodyBuilder) SetInstanceAgentDetails(attr map[string]interface{}) ServiceBuilder {
	b.serviceBody.InstanceAgentDetails = attr
	return b
}

func (b *serviceBodyBuilder) SetRevisionAgentDetails(attr map[string]interface{}) ServiceBuilder {
	b.serviceBody.RevisionAgentDetails = attr
	return b
}

func (b *serviceBodyBuilder) SetServiceEndpoints(endpoints []EndpointDefinition) ServiceBuilder {
	b.serviceBody.Endpoints = endpoints
	return b
}

func (b *serviceBodyBuilder) AddServiceEndpoint(protocol, host string, port int32, basePath string) ServiceBuilder {
	ep := EndpointDefinition{
		Host:     host,
		Port:     port,
		Protocol: protocol,
		BasePath: basePath,
	}
	b.serviceBody.Endpoints = append(b.serviceBody.Endpoints, ep)
	return b
}

func (b *serviceBodyBuilder) SetUnstructuredType(assetType string) ServiceBuilder {
	b.serviceBody.UnstructuredProps.AssetType = assetType
	return b
}

func (b *serviceBodyBuilder) SetUnstructuredContentType(contentType string) ServiceBuilder {
	b.serviceBody.UnstructuredProps.ContentType = contentType
	return b
}

func (b *serviceBodyBuilder) SetUnstructuredLabel(label string) ServiceBuilder {
	b.serviceBody.UnstructuredProps.Label = label
	return b
}

func (b *serviceBodyBuilder) SetUnstructuredFilename(filename string) ServiceBuilder {
	b.serviceBody.UnstructuredProps.Filename = filename
	return b
}

func (b *serviceBodyBuilder) SetTeamName(teamName string) ServiceBuilder {
	b.serviceBody.TeamName = teamName
	return b
}

func (b *serviceBodyBuilder) Build() (ServiceBody, error) {
	if b.err != nil {
		return b.serviceBody, b.err
	}

	specParser := NewSpecResourceParser(b.serviceBody.SpecDefinition, b.serviceBody.ResourceType)
	err := specParser.Parse()
	if err != nil {
		return b.serviceBody, fmt.Errorf("failed to parse service specification for '%s': %s", b.serviceBody.APIName, err)
	}
	specProcessor := specParser.GetSpecProcessor()
	if b.serviceBody.ResourceContentType == "" {
		b.serviceBody.ResourceContentType = specParser.getResourceContentType()
	}
	b.serviceBody.ResourceType = specProcessor.GetResourceType()
	b.serviceBody.specHash = fmt.Sprintf("%v", specParser.specHash)
	b.serviceBody.specVersion = specProcessor.GetVersion()
	if b.serviceBody.specVersion == "" {
		b.serviceBody.specVersion = b.serviceBody.Version
	}

	// Check if the type is unstructured to gather more info

	if len(b.serviceBody.Endpoints) == 0 {
		endPoints, err := specProcessor.GetEndpoints()
		if err != nil {
			return b.serviceBody, fmt.Errorf("failed to create endpoints for '%s': %s", b.serviceBody.APIName, err)
		}
		b.serviceBody.Endpoints = endPoints
	}

	// check all endpoints paths are valid
	for _, ep := range b.serviceBody.Endpoints {
		if ep.BasePath != "" && !pathRe.MatchString(ep.BasePath) {
			return b.serviceBody, fmt.Errorf("path did not validate against '^/' for '%s'", ep.BasePath)
		}
	}

	if b.serviceBody.dataplaneType == "" {
		b.serviceBody.dataplaneType = "Unidentified"
	}

	var i interface{} = specProcessor
	val, ok := i.(OasSpecProcessor)
	if !ok {
		return b.serviceBody, nil
	}

	val.ParseAuthInfo()

	// get the auth policy from the spec
	b.serviceBody.authPolicies = val.GetAuthPolicies()

	// use the first auth policy in the list as the AuthPolicy for determining if subscriptions are enabled
	if len(b.serviceBody.authPolicies) > 0 {
		b.serviceBody.AuthPolicy = b.serviceBody.authPolicies[0]
	}

	// get the apikey info
	b.serviceBody.apiKeyInfo = val.GetAPIKeyInfo()

	// get oauth scopes
	b.serviceBody.scopes = val.GetOAuthScopes()

	// if the spec has multiple then use the oauth ard
	err = b.serviceBody.createAccessRequestDefinition()
	if err != nil {
		return b.serviceBody, err
	}

	// only set ard name based on spec if not already set, use first auth we find
	if b.serviceBody.ardName != "" {
		return b.serviceBody, nil
	}

	if b.serviceBody.ignoreSpecBasesCreds {
		return b.serviceBody, nil
	}

	for _, p := range b.serviceBody.authPolicies {
		if p == Basic {
			b.serviceBody.ardName = provisioning.BasicAuthARD
			break
		}
		if p == Apikey {
			b.serviceBody.ardName = provisioning.APIKeyARD
			break
		}
	}

	return b.serviceBody, nil
}

// SetCredentialRequestDefinitions -
func (b *serviceBodyBuilder) SetCredentialRequestDefinitions(credentialRequestDefNames []string) ServiceBuilder {
	b.serviceBody.credentialRequestPolicies = credentialRequestDefNames
	return b
}

// AddCredentialRequestDefinition -
func (b *serviceBodyBuilder) AddCredentialRequestDefinition(credentialRequestDefName string) ServiceBuilder {
	b.serviceBody.credentialRequestPolicies = append(b.serviceBody.credentialRequestPolicies, credentialRequestDefName)
	return b
}

// SetAccessRequestDefinitionName -
func (b *serviceBodyBuilder) SetAccessRequestDefinitionName(accessRequestDefName string, isUnique bool) ServiceBuilder {
	b.serviceBody.ardName = accessRequestDefName
	b.serviceBody.uniqueARD = isUnique
	return b
}

func (b *serviceBodyBuilder) SetIgnoreSpecBasedCreds(ignore bool) ServiceBuilder {
	b.serviceBody.ignoreSpecBasesCreds = ignore
	return b
}

func (b *serviceBodyBuilder) SetSourceDataplaneType(dataplaneType DataplaneType, isDesign bool) ServiceBuilder {
	b.serviceBody.dataplaneType = dataplaneType
	b.serviceBody.isDesignDataplane = isDesign
	return b
}

func (b *serviceBodyBuilder) SetReferenceServiceName(serviceName, envName string) ServiceBuilder {
	if envName != "" {
		b.serviceBody.referencedServiceName = fmt.Sprintf("%s/%s", envName, serviceName)
	} else {
		b.serviceBody.referencedServiceName = serviceName
	}

	return b
}

func (b *serviceBodyBuilder) SetReferenceInstanceName(instanceName, envName string) ServiceBuilder {
	if envName != "" {
		b.serviceBody.referencedInstanceName = fmt.Sprintf("%s/%s", envName, instanceName)
	} else {
		b.serviceBody.referencedInstanceName = instanceName
	}
	return b
}
