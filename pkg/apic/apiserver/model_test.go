package models

import (
	"encoding/json"
	"testing"
	"time"

	m "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/stretchr/testify/assert"
)

// should handle marshaling and unmarshalling for an apiserver resource with a custom sub resource
func TestAPIServiceMarshal(t *testing.T) {
	activityTime := time.Now()
	newV1Time := v1.Time(activityTime)

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
		Status: m.ApiServiceStatus{
			Phase: m.ApiServiceStatusPhase{
				Name:           "status",
				Level:          "ok",
				Message:        "",
				TransitionTime: newV1Time,
			},
		},
	}

	bts, err := json.Marshal(svc1)
	assert.Nil(t, err)
	assert.NotNil(t, bts)

	svc2 := &m.APIService{
		Status: m.ApiServiceStatus{
			Phase: m.ApiServiceStatusPhase{
				Name:           "status",
				Level:          "ok",
				Message:        "",
				TransitionTime: newV1Time,
			},
		},
	}

	err = json.Unmarshal(bts, svc2)
	assert.Nil(t, err)

	// override the audit metadata to easily assert the two structs are equal
	svc1.Metadata.Audit = apiv1.AuditMetadata{}
	svc2.Metadata.Audit = apiv1.AuditMetadata{}
	assert.Equal(t, svc1, svc2)
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

// Should create an APIService from a ResourceInstance
func TestAPIServiceFromInstance(t *testing.T) {
	// convert a service to an instance
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
	ri1, err := svc1.AsInstance()
	assert.Nil(t, err)

	// call FromInstance using the first service, which should fill all the fields of svc2 from svc1
	svc2 := &m.APIService{}
	err = svc2.FromInstance(ri1)
	assert.Nil(t, err)

	// the api services should be equal, and their resource instances should be equal
	ri2, err := svc2.AsInstance()
	assert.Nil(t, err)
	assert.Equal(t, ri1, ri2)

	svc1.Metadata.Audit = apiv1.AuditMetadata{}
	svc2.Metadata.Audit = apiv1.AuditMetadata{}
	assert.Equal(t, svc1, svc2)
}

// should unmarshal a populated governance agent into an empty governance agent.
// Unmarshalling should handle the pre-defined sub resources and any dynamic sub resources.
func TestGovernanceAgentResource(t *testing.T) {
	gov1 := &m.GovernanceAgent{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Group: "management",
					Kind:  "GovernanceAgent",
				},
				APIVersion: "v1alpha1",
			},
			Name:  "name",
			Title: "title",
			Metadata: apiv1.Metadata{
				ID: "123",
			},
		},
		Agentconfigstatus: m.GovernanceAgentAgentconfigstatus{
			ResourceVersion: "321",
			ErrorMessage:    "msg",
			ConfigState:     "state",
		},
		Spec: m.GovernanceAgentSpec{
			DataplaneType: "aws",
			Config: map[string]interface{}{
				"abc": "123",
			},
		},
		Status: m.GovernanceAgentStatus{
			Version:    "v1",
			SdkVersion: "1.0.0",
		},
	}
	gov1.SetSubResource("x-agent-details", map[string]interface{}{
		"abc": "1223",
	})
	bts, err := json.Marshal(gov1)
	assert.Nil(t, err)
	assert.NotNil(t, bts)

	gov2 := &m.GovernanceAgent{}

	err = json.Unmarshal(bts, gov2)
	assert.Nil(t, err)

	// expect that the sub resources defined on the gov1 resource are contained in the SubResource map
	assert.Contains(t, gov2.SubResources, "x-agent-details")
	// SubResources should not contain any field already defined on the resource
	assert.NotContains(t, gov2.SubResources, "status")
	assert.NotContains(t, gov2.SubResources, "agentconfigstatus")

	// expect that the two resources contain the same data when marshalled into bytes
	ri1, err := gov1.AsInstance()
	assert.Nil(t, err)

	ri2, err := gov2.AsInstance()
	assert.Nil(t, err)

	assert.Equal(t, ri1.GetRawResource(), ri2.GetRawResource())
}
