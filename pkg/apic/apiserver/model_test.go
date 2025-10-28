package models

import (
	"encoding/json"
	"testing"
	"time"

	m "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/stretchr/testify/assert"
)

// should handle marshaling and unmarshalling for an apiserver resource with a custom sub resource
func TestAPIServiceMarshal(t *testing.T) {
	svc1 := &m.APIService{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1.GroupVersionKind{
				GroupKind:  v1.GroupKind{Group: "management", Kind: "APIService"},
				APIVersion: "v1",
			},
			Name:  "name",
			Title: "title",
			Metadata: v1.Metadata{
				ID:              "123",
				Audit:           v1.AuditMetadata{},
				ResourceVersion: "1",
				SelfLink:        "/self/link",
				State:           "state",
			},
			Attributes: map[string]string{
				"attr1": "val1",
				"attr2": "val2",
			},
			Tags: []string{"tag1", "tag2"},
			Finalizers: []v1.Finalizer{
				{Name: "finalizer1"},
				{Name: "finalizer2"},
			},
			SubResources: map[string]interface{}{
				"x-agent-details": map[string]interface{}{
					"x-agent-id": "123",
				},
			},
		},
		Owner: &v1.Owner{
			Type: v1.TeamOwner,
			ID:   "233",
		},
		Spec: m.ApiServiceSpec{
			Description: "desc",
			Icon: m.ApiServiceSpecIcon{
				ContentType: "image/png",
				Data:        "data",
			},
		},
		Status: &v1.ResourceStatus{
			Level: "Error",
			Reasons: []v1.ResourceStatusReason{
				{
					Type:      "Error",
					Detail:    "",
					Timestamp: getTimestamp(),
				},
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
	svc1.Metadata.Audit = v1.AuditMetadata{}
	svc2.Metadata.Audit = v1.AuditMetadata{}
	assert.Equal(t, svc1, svc2)
}

// should unmarshal when owner is not set
func TestAPIServiceMarshalNoOwner(t *testing.T) {
	svc1 := &m.APIService{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1.GroupVersionKind{
				GroupKind:  v1.GroupKind{Group: "management", Kind: "APIService"},
				APIVersion: "v1",
			},
			Name:  "name",
			Title: "title",
			Metadata: v1.Metadata{
				ID: "123",
			},
			Finalizers: []v1.Finalizer{
				{Name: "finalizer1"},
				{Name: "finalizer2"},
			},
			SubResources: map[string]interface{}{
				"x-agent-details": map[string]interface{}{
					"x-agent-id": "123",
				},
			},
		},
		Spec: m.ApiServiceSpec{
			Description: "desc",
		},
		Status: &v1.ResourceStatus{
			Level: "Error",
			Reasons: []v1.ResourceStatusReason{
				{
					Type:      "Error",
					Detail:    "",
					Timestamp: getTimestamp(),
				},
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
	svc1.Metadata.Audit = v1.AuditMetadata{}
	svc2.Metadata.Audit = v1.AuditMetadata{}
	assert.Equal(t, svc1, svc2)
}

// should convert an APIService to a ResourceInstance
func TestAPIServiceAsInstance(t *testing.T) {
	newTime := getTimestamp()
	svc := &m.APIService{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1.GroupVersionKind{
				GroupKind:  v1.GroupKind{Group: "management", Kind: "APIService"},
				APIVersion: "v1",
			},
			Name:  "name",
			Title: "title",
			Metadata: v1.Metadata{
				ID:              "123",
				Audit:           v1.AuditMetadata{},
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
				"agentdetails": map[string]interface{}{},
				"details":      map[string]interface{}{},
				"profile":      map[string]interface{}{},
				"x-agent-details": map[string]interface{}{
					"x-agent-id": "123",
				},
				"status": map[string]interface{}{
					"level": "Error",
					"reasons": []interface{}{
						map[string]interface{}{
							"type":      "Error",
							"detail":    "error",
							"timestamp": time.Time(newTime).Format(v1.APIServerTimeFormat),
						},
					},
				},
				"references": map[string]interface{}{},
			},
		},
		Owner: &v1.Owner{
			Type: v1.TeamOwner,
			ID:   "233",
		},
		Spec: m.ApiServiceSpec{
			Description: "desc",
			Icon: m.ApiServiceSpecIcon{
				ContentType: "image/png",
				Data:        "data",
			},
		},
		Status: &v1.ResourceStatus{
			Level: "Error",
			Reasons: []v1.ResourceStatusReason{
				{
					Type:      "Error",
					Detail:    "error",
					Timestamp: newTime,
				},
			},
		},
	}

	ri, err := svc.AsInstance()
	assert.Nil(t, err)

	// override the audit metadata to easily assert the two structs are equal
	svc.Metadata.Audit = v1.AuditMetadata{}
	ri.Metadata.Audit = v1.AuditMetadata{}

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
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1.GroupVersionKind{
				GroupKind:  v1.GroupKind{Group: "management", Kind: "APIService"},
				APIVersion: "v1",
			},
			Name:  "name",
			Title: "title",
			Metadata: v1.Metadata{
				ID:              "123",
				Audit:           v1.AuditMetadata{},
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
		Owner: &v1.Owner{
			Type: v1.TeamOwner,
			ID:   "233",
		},
		Spec: m.ApiServiceSpec{
			Description: "desc",
			Icon: m.ApiServiceSpecIcon{
				ContentType: "image/png",
				Data:        "data",
			},
		},
		Status: &v1.ResourceStatus{
			Level: "Success",
			Reasons: []v1.ResourceStatusReason{
				{
					Type:      "Error",
					Detail:    "",
					Timestamp: getTimestamp(),
				},
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

	svc1.Metadata.Audit = v1.AuditMetadata{}
	svc2.Metadata.Audit = v1.AuditMetadata{}
	assert.Equal(t, svc1, svc2)
}

// getTimestamp - Returns current timestamp formatted for API Server
func getTimestamp() v1.Time {
	activityTime := time.Now()
	newV1Time := v1.Time(activityTime)

	// marshall the time in and out of JSON to get same format
	timeBytes, _ := newV1Time.MarshalJSON()
	newV1Time.UnmarshalJSON(timeBytes)
	return newV1Time
}
