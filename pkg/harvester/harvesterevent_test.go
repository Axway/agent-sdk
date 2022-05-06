package harvester

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestHarvesterEventConversion(t *testing.T) {
	harvesterEvent := &resourceEntryExternalEvent{
		ID:            "123",
		Time:          "2021-11-30 11:59:25.01",
		Version:       "v1",
		Product:       "AmplifyCentral",
		CorrelationID: "441c85fc-b4cd-46fe-aae2-5aaf6ef86b8e",
		Organization: &proto.Organization{
			Id: "224879455212557",
		},
		Type: "ResourceCreated",
		Payload: &harvesterResourceInstance{
			Group: "management",
			Kind:  "APIServiceInstance",
			Name:  "test",
			Attributes: map[string]string{
				"createdBy": "DiscoveryAgent",
			},
			Metadata: &harvesterResourceMetadata{
				ID:       "12345",
				SelfLink: "/management/v1alpha1/environments/sample/apiserviceinstances/test",
				References: []*harvesterResourceReference{
					{
						ID:        "8ac9934a7d6f94aa017d70b6bc2204dd",
						Kind:      "APIServiceRevision",
						Name:      "test",
						ScopeKind: "Environment",
						ScopeName: "sample",
						SelfLink:  "/management/v1alpha1/environments/sample/apiservicerevisions/test",
						Type:      "HARD",
					},
				},
				Scope: &proto.Metadata_ScopeKind{
					Id:       "123456",
					Kind:     "Environment",
					Name:     "sample",
					SelfLink: "/management/v1alpha1/environments/sample",
				},
			},
		},
		Metadata: &proto.EventMeta{
			WatchTopicID:       "1234",
			WatchTopicSelfLink: "/management/v1alpha1/watchtopics/agent-watch",
			SequenceID:         100,
		},
	}

	event := harvesterEvent.toProtoEvent()

	assert.Equal(t, harvesterEvent.ID, event.Id)
	assert.Equal(t, harvesterEvent.Time, event.Time)
	assert.Equal(t, harvesterEvent.Version, event.Version)
	assert.Equal(t, harvesterEvent.Product, event.Product)
	assert.Equal(t, harvesterEvent.CorrelationID, event.CorrelationId)
	assert.Equal(t, proto.Event_CREATED, event.Type)

	assert.NotNil(t, event.Organization)
	assert.Equal(t, harvesterEvent.Organization.Id, event.Organization.Id)

	assert.NotNil(t, event.Payload)
	assert.Equal(t, harvesterEvent.Payload.Group, event.Payload.Group)
	assert.Equal(t, harvesterEvent.Payload.Kind, event.Payload.Kind)
	assert.Equal(t, harvesterEvent.Payload.Name, event.Payload.Name)

	assert.NotNil(t, event.Payload.Attributes)
	assert.Equal(t, harvesterEvent.Payload.Attributes, event.Payload.Attributes)

	assert.NotNil(t, event.Payload.Metadata)
	assert.Equal(t, harvesterEvent.Payload.Metadata.ID, event.Payload.Metadata.Id)
	assert.Equal(t, harvesterEvent.Payload.Metadata.SelfLink, event.Payload.Metadata.SelfLink)
	assert.NotNil(t, harvesterEvent.Payload.Metadata.Scope)
	assert.Equal(t, harvesterEvent.Payload.Metadata.Scope.Id, event.Payload.Metadata.Scope.Id)
	assert.Equal(t, harvesterEvent.Payload.Metadata.Scope.Kind, event.Payload.Metadata.Scope.Kind)
	assert.Equal(t, harvesterEvent.Payload.Metadata.Scope.Name, event.Payload.Metadata.Scope.Name)
	assert.Equal(t, harvesterEvent.Payload.Metadata.Scope.SelfLink, event.Payload.Metadata.Scope.SelfLink)

	assert.NotNil(t, event.Payload.Metadata.References)
	assert.Equal(t, len(harvesterEvent.Payload.Metadata.References), len(event.Payload.Metadata.References))
	assert.Equal(t, harvesterEvent.Payload.Metadata.References[0].ID, event.Payload.Metadata.References[0].Id)
	assert.Equal(t, harvesterEvent.Payload.Metadata.References[0].Kind, event.Payload.Metadata.References[0].Kind)
	assert.Equal(t, harvesterEvent.Payload.Metadata.References[0].Name, event.Payload.Metadata.References[0].Name)
	assert.Equal(t, harvesterEvent.Payload.Metadata.References[0].ScopeKind, event.Payload.Metadata.References[0].ScopeKind)
	assert.Equal(t, harvesterEvent.Payload.Metadata.References[0].ScopeName, event.Payload.Metadata.References[0].ScopeName)
	assert.Equal(t, harvesterEvent.Payload.Metadata.References[0].SelfLink, event.Payload.Metadata.References[0].SelfLink)
	assert.Equal(t, proto.Reference_HARD, event.Payload.Metadata.References[0].Type)

	assert.NotNil(t, event.Metadata)
	assert.Equal(t, harvesterEvent.Metadata.WatchTopicID, event.Metadata.WatchTopicID)
	assert.Equal(t, harvesterEvent.Metadata.WatchTopicSelfLink, event.Metadata.WatchTopicSelfLink)
	assert.Equal(t, harvesterEvent.Metadata.SequenceID, event.Metadata.SequenceID)
}
