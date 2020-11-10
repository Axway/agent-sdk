package apic

import (
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
)

// ServiceBuilder - Interface to build the service body
type ServiceBuilder interface {
	SetID(ID string) ServiceBuilder
	SetTitle(title string) ServiceBuilder
	SetAPIName(apiName string) ServiceBuilder
	SetURL(url string) ServiceBuilder
	SetStage(stage string) ServiceBuilder
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
			AuthPolicy:        Passthrough,
			ResourceType:      Oas3,
			CreatedBy:         config.AgentTypeName,
			State:             PublishedStatus,
			Status:            PublishedStatus,
			ServiceAttributes: make(map[string]string),
		},
	}

}
func (b *serviceBodyBuilder) SetID(ID string) ServiceBuilder {
	b.serviceBody.RestAPIID = ID
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

func (b *serviceBodyBuilder) SetDescription(description string) ServiceBuilder {
	b.serviceBody.Description = description
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
	b.serviceBody.Swagger = spec
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

func (b *serviceBodyBuilder) Build() (ServiceBody, error) {
	return b.serviceBody, b.err
}
