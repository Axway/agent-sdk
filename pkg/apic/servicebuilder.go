package apic

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/config"
)

const (
	maxDescriptionLength = 350
	strEllipsis          = "..."
)

// ServiceBuilder - Interface to build the service body
type ServiceBuilder interface {
	SetID(ID string) ServiceBuilder
	SetPrimaryKey(key string) ServiceBuilder
	SetTitle(title string) ServiceBuilder
	SetAPIName(apiName string) ServiceBuilder
	SetURL(url string) ServiceBuilder
	SetStage(stage string) ServiceBuilder
	SetStageDescriptor(stageDescriptor string) ServiceBuilder
	SetDescription(description string) ServiceBuilder
	SetVersion(version string) ServiceBuilder
	SetAuthPolicy(authPolicy string) ServiceBuilder
	SetAPISpec(spec []byte) ServiceBuilder
	SetDocumentation(documentation []byte) ServiceBuilder
	SetTags(tags map[string]interface{}) ServiceBuilder
	SetImage(image string) ServiceBuilder
	SetImageContentType(imageContentType string) ServiceBuilder
	SetResourceType(resourceType string) ServiceBuilder
	SetAltRevisionPrefix(revisionPrefix string) ServiceBuilder
	SetSubscriptionName(subscriptionName string) ServiceBuilder
	SetAPIUpdateSeverity(apiUpdateSeverity string) ServiceBuilder
	SetState(state string) ServiceBuilder
	SetStatus(status string) ServiceBuilder
	SetServiceAttribute(serviceAttribute map[string]string) ServiceBuilder
	SetInstanceAttribute(instanceAttribute map[string]string) ServiceBuilder
	SetRevisionAttribute(revisionAttribute map[string]string) ServiceBuilder
	SetServiceEndpoints(endpoints []EndpointDefinition) ServiceBuilder
	AddServiceEndpoint(protocol, host string, port int32, basePath string) ServiceBuilder

	SetUnstructuredType(assetType string) ServiceBuilder
	SetUnstructuredContentType(contentType string) ServiceBuilder
	SetUnstructuredLabel(label string) ServiceBuilder
	SetUnstructuredFilename(filename string) ServiceBuilder
	SetTeamName(teamName string) ServiceBuilder
	SetCategories(categories []string) ServiceBuilder

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
			AuthPolicy:         Passthrough,
			authPolicies:       make([]string, 0),
			CreatedBy:          config.AgentTypeName,
			State:              PublishedStatus,
			Status:             PublishedStatus,
			ServiceAttributes:  make(map[string]string),
			RevisionAttributes: make(map[string]string),
			InstanceAttributes: make(map[string]string),
			StageDescriptor:    "Stage",
			Endpoints:          make([]EndpointDefinition, 0),
			UnstructuredProps:  &UnstructuredProperties{},
			categoryTitles:     make([]string, 0),
			categoryNames:      make([]string, 0),
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

func (b *serviceBodyBuilder) SetServiceAttribute(serviceAttribute map[string]string) ServiceBuilder {
	b.serviceBody.ServiceAttributes = serviceAttribute
	return b
}

func (b *serviceBodyBuilder) SetInstanceAttribute(instanceAttribute map[string]string) ServiceBuilder {
	b.serviceBody.InstanceAttributes = instanceAttribute
	return b
}

func (b *serviceBodyBuilder) SetRevisionAttribute(revisionAttribute map[string]string) ServiceBuilder {
	b.serviceBody.RevisionAttributes = revisionAttribute
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

func (b *serviceBodyBuilder) SetAltRevisionPrefix(revisionPrefix string) ServiceBuilder {
	b.serviceBody.AltRevisionPrefix = revisionPrefix
	return b
}

func (b *serviceBodyBuilder) SetTeamName(teamName string) ServiceBuilder {
	b.serviceBody.TeamName = teamName
	return b
}

func (b *serviceBodyBuilder) SetCategories(categories []string) ServiceBuilder {
	b.serviceBody.categoryTitles = categories
	return b
}

func (b *serviceBodyBuilder) Build() (ServiceBody, error) {
	if b.err != nil {
		return b.serviceBody, b.err
	}

	specParser := newSpecResourceParser(b.serviceBody.SpecDefinition, b.serviceBody.ResourceType)
	err := specParser.parse()
	if err != nil {
		return b.serviceBody, fmt.Errorf("failed to parse service specification for '%s': %s", b.serviceBody.APIName, err)
	}
	specProcessor := specParser.getSpecProcessor()
	b.serviceBody.ResourceType = specProcessor.getResourceType()

	// Check if the type is unstructured to gather more info

	if len(b.serviceBody.Endpoints) == 0 {
		endPoints, err := specProcessor.getEndpoints()
		if err != nil {
			return b.serviceBody, fmt.Errorf("failed to create endpoints for '%s': %s", b.serviceBody.APIName, err)
		}
		b.serviceBody.Endpoints = endPoints
	}

	var i interface{} = specProcessor
	if val, ok := i.(oasSpecProcessor); ok {
		// get the auth policy from the spec
		b.serviceBody.authPolicies, b.serviceBody.apiKeyInfo = val.getAuthInfo()

		// use the first auth policy in the list as the AuthPolicy for determining if subscriptions are enabled
		if len(b.serviceBody.authPolicies) > 0 {
			b.serviceBody.AuthPolicy = b.serviceBody.authPolicies[0]
		}
	}

	return b.serviceBody, nil
}
