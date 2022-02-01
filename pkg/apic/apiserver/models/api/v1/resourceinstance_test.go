package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// should marshal bytes of a ResourceInstance into a new empty ResourceInstance
func TestResourceInstanceMarshalJSON(t *testing.T) {
	r1 := &ResourceInstance{
		ResourceMeta: ResourceMeta{
			GroupVersionKind: GroupVersionKind{
				APIVersion: "v1",
				GroupKind: GroupKind{
					Group: "group",
					Kind:  "kind",
				},
			},
			Name:     "name",
			Title:    "title",
			Metadata: Metadata{},
			Attributes: map[string]string{
				"key": "value",
			},
			Tags: []string{"tag1", "tag2"},
			SubResources: map[string]interface{}{
				"subresource": map[string]interface{}{
					"sub1": "value",
				},
			},
		},
		Owner: &Owner{
			Type: TeamOwner,
			ID:   "123",
		},
		Spec: map[string]interface{}{
			"description": "test",
		},
		rawResource: nil,
	}

	bts, err := json.Marshal(r1)
	assert.Nil(t, err)
	assert.NotNil(t, bts)

	r2 := &ResourceInstance{}

	err = json.Unmarshal(bts, r2)
	assert.Equal(t, json.RawMessage(bts), r2.GetRawResource())
	assert.Equal(t, r1.Spec, r2.Spec)
	assert.Equal(t, r1.Owner, r2.Owner)

	r1.Metadata.Audit = AuditMetadata{}
	r2.Metadata.Audit = AuditMetadata{}
	assert.Equal(t, r1.ResourceMeta, r2.ResourceMeta)
}

func TestResourceInstance_FromInstance(t *testing.T) {
	ri1 := &ResourceInstance{
		ResourceMeta: ResourceMeta{
			GroupVersionKind: GroupVersionKind{
				APIVersion: "v1",
				GroupKind: GroupKind{
					Group: "group",
					Kind:  "kind",
				},
			},
			Name:     "name",
			Title:    "title",
			Metadata: Metadata{},
			Attributes: map[string]string{
				"key": "value",
			},
			Tags: []string{"tag1", "tag2"},
			SubResources: map[string]interface{}{
				"subresource": map[string]interface{}{
					"sub1": "value",
				},
			},
		},
		Owner: &Owner{
			Type: TeamOwner,
			ID:   "123",
		},
		Spec: map[string]interface{}{
			"description": "test",
		},
	}

	ri2 := &ResourceInstance{}

	inst, err := ri1.AsInstance()
	assert.Nil(t, err)
	err = ri2.FromInstance(inst)
	assert.Nil(t, err)
	assert.Equal(t, ri1, ri2)
}
