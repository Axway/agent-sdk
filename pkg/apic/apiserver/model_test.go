package models

import (
	"encoding/json"
	"testing"

	m "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/stretchr/testify/assert"
)

// should handle marshaling and unmarshalling for an apiserver resource with a custom sub resource
func TestAPIServiceMarshal(t *testing.T) {
	svc1 := &m.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind:  apiv1.GroupKind{Group: "management", Kind: "APIService"},
				APIVersion: "v1",
			},
			Name:  "name",
			Title: "title",
			Metadata: apiv1.Metadata{
				ID:              "123",
				Audit:           apiv1.AuditMetadata{},
				ResourceVersion: "1",
				SelfLink:        "/self/link",
				State:           "state",
			},
			Attributes: map[string]string{
				"attr1": "val1",
				"attr2": "val2",
			},
			Tags: []string{"tag1", "tag2"},
			Finalizers: []apiv1.Finalizer{
				{Name: "finalizer1"},
				{Name: "finalizer2"},
			},
			SubResources: map[string]interface{}{
				"x-agent-details": map[string]interface{}{
					"x-agent-id": "123",
				},
			},
		},
		Owner: &apiv1.Owner{
			Type: apiv1.TeamOwner,
			ID:   "233",
		},
		Spec: m.ApiServiceSpec{
			Description: "desc",
			Categories:  []string{"cat1", "cat2"},
			Icon: m.ApiServiceSpecIcon{
				ContentType: "image/png",
				Data:        "data",
			},
		},
	}

	bts, err := json.Marshal(svc1)
	assert.Nil(t, err)
	assert.NotNil(t, bts)

	svc2 := &m.APIService{}

	err = json.Unmarshal(bts, svc2)
	assert.Nil(t, err)

	// override the audit metadata to easily assert the two structs are equal
	svc1.Metadata.Audit = apiv1.AuditMetadata{}
	svc2.Metadata.Audit = apiv1.AuditMetadata{}
	assert.Equal(t, svc1, svc2)

	svc3 := &m.APIService{}

	// should return an error when given an invalid spec
	b := []byte(`{"spec":"def"}`)
	err = json.Unmarshal(b, svc3)
	assert.NotNil(t, err)

	// should return an error when given an invalid owner
	b = []byte(`{"owner":"def"}`)
	err = json.Unmarshal(b, svc3)
	assert.NotNil(t, err)
}

// should unmarshal when owner is not set
func TestAPIServiceMarshalNoOwner(t *testing.T) {
	svc1 := &m.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind:  apiv1.GroupKind{Group: "management", Kind: "APIService"},
				APIVersion: "v1",
			},
			Name:  "name",
			Title: "title",
			Metadata: apiv1.Metadata{
				ID: "123",
			},
		},
		Spec: m.ApiServiceSpec{
			Description: "desc",
			Categories:  []string{"cat1", "cat2"},
		},
	}

	bts, err := json.Marshal(svc1)
	assert.Nil(t, err)
	assert.NotNil(t, bts)

	svc2 := &m.APIService{}

	err = json.Unmarshal(bts, svc2)
	assert.Nil(t, err)

	// override the audit metadata to easily assert the two structs are equal
	svc1.Metadata.Audit = apiv1.AuditMetadata{}
	svc2.Metadata.Audit = apiv1.AuditMetadata{}
	assert.Equal(t, svc1, svc2)
}

// should convert an APIService to a ResourceInstance
func TestAPIServiceAsInstance(t *testing.T) {
	svc := &m.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind:  apiv1.GroupKind{Group: "management", Kind: "APIService"},
				APIVersion: "v1",
			},
			Name:  "name",
			Title: "title",
			Metadata: apiv1.Metadata{
				ID:              "123",
				Audit:           apiv1.AuditMetadata{},
				ResourceVersion: "1",
				SelfLink:        "/self/link",
				State:           "state",
			},
			Attributes: map[string]string{
				"attr1": "val1",
				"attr2": "val2",
			},
			Tags:       []string{"tag1", "tag2"},
			Finalizers: nil,
			SubResources: map[string]interface{}{
				"x-agent-details": map[string]interface{}{
					"x-agent-id": "123",
				},
			},
		},
		Owner: &apiv1.Owner{
			Type: apiv1.TeamOwner,
			ID:   "233",
		},
		Spec: m.ApiServiceSpec{
			Description: "desc",
			Categories:  []string{"cat1", "cat2"},
			Icon: m.ApiServiceSpecIcon{
				ContentType: "image/png",
				Data:        "data",
			},
		},
	}

	ri, err := svc.AsInstance()
	assert.Nil(t, err)

	// override the audit metadata to easily assert the two structs are equal
	svc.Metadata.Audit = apiv1.AuditMetadata{}
	ri.Metadata.Audit = apiv1.AuditMetadata{}

	// marshal the instance spec to bytes, then convert it to an ApiServiceSpec
	// to see if it matches the svc.Spec field
	bts, err := json.Marshal(ri.Spec)
	assert.Nil(t, err)

	instSpec := &m.ApiServiceSpec{}
	err = json.Unmarshal(bts, instSpec)
	assert.Nil(t, err)

	assert.Equal(t, svc.Spec, *instSpec)
	assert.Equal(t, svc.Owner, ri.Owner)
	assert.Equal(t, svc.ResourceMeta, ri.ResourceMeta)

	svcBytes, err := json.Marshal(svc)
	assert.Nil(t, err)

	assert.Equal(t, json.RawMessage(svcBytes), ri.GetRawResource())
}

func TestDiscoveryAgentResource(t *testing.T) {
	t.Skip()
	disc1 := &m.DiscoveryAgent{
		ResourceMeta: apiv1.ResourceMeta{},
		Owner:        nil,
		Spec: m.DiscoveryAgentSpec{
			DataplaneType: "abc",
			Config: m.DiscoveryAgentSpecConfig{
				Filter:     "123",
				OwningTeam: "aa",
			},
		},
		Status: m.DiscoveryAgentStatus{
			Version:                "1",
			LatestAvailableVersion: "1",
			State:                  "running",
			PreviousState:          "failed",
		},
	}

	bts, err := json.Marshal(disc1)
	assert.Nil(t, err)
	assert.NotNil(t, bts)

	disc2 := &m.DiscoveryAgent{}

	err = json.Unmarshal(bts, disc2)
	assert.Nil(t, err)

	// override the audit metadata to easily assert the two structs are equal
	disc1.Metadata.Audit = apiv1.AuditMetadata{}
	disc2.Metadata.Audit = apiv1.AuditMetadata{}
	assert.Equal(t, disc1, disc2)
}
