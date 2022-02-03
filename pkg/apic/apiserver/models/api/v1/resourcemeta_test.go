package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceMetaMarshal(t *testing.T) {
	apiID := "99"
	primaryKey := "4321"

	meta1 := &ResourceMeta{
		GroupVersionKind: GroupVersionKind{
			GroupKind: GroupKind{
				Group: "group",
				Kind:  "kind",
			},
			APIVersion: "v1",
		},
		Name:  "meta1",
		Title: "meta1",
		Metadata: Metadata{
			ID: "123",
		},
		Tags: []string{"tag1", "tag2"},
		Attributes: map[string]string{
			"abc": "123",
		},
	}

	empty := meta1.GetSubResource("abc123")
	assert.Nil(t, empty)

	meta1.SetSubResource("x-agent-details", map[string]interface{}{
		"apiID": apiID,
	})

	// Get the sub resource by name, and update the value returned
	resource := meta1.GetSubResource("x-agent-details")
	m := resource.(map[string]interface{})
	m["primaryKey"] = primaryKey

	// save the resource with the new value
	meta1.SetSubResource("x-agent-details", m)

	bts, err := json.Marshal(meta1)
	assert.Nil(t, err)
	assert.NotNil(t, bts)

	// Unmarshal the ResourceMeta bytes to a map to confirm that MarshalJSON saved the SubResource correctly
	values := map[string]interface{}{}
	err = json.Unmarshal(bts, &values)
	assert.Nil(t, err)

	// Get the x-agent-details sub resource, and convert it to a map
	subResource := values["x-agent-details"]
	xAgentDetailsSub, ok := subResource.(map[string]interface{})
	assert.True(t, ok)

	assert.Equal(t, apiID, xAgentDetailsSub["apiID"])
	assert.Equal(t, primaryKey, xAgentDetailsSub["primaryKey"])

	// Unmarshal the data from meta1, to meta2, which is empty, and assert meta2 contains the same data from meta1
	meta2 := &ResourceMeta{}
	err = json.Unmarshal(bts, meta2)
	assert.Nil(t, err)

	meta1.Metadata.Audit = AuditMetadata{}
	meta2.Metadata.Audit = AuditMetadata{}
	assert.Equal(t, meta1, meta2)

	// expect to the sub resources to be equal
	assert.True(t, len(meta2.SubResources) == 1)
	assert.Equal(t, xAgentDetailsSub, meta2.SubResources["x-agent-details"])
}

func TestResourceMeta(t *testing.T) {
	meta := &ResourceMeta{
		GroupVersionKind: GroupVersionKind{
			GroupKind: GroupKind{
				Group: "group",
				Kind:  "kind",
			},
			APIVersion: "v1",
		},
		Title: "title",
		Metadata: Metadata{
			ID: "333",
		},
	}

	assert.Equal(t, meta.Metadata, meta.GetMetadata())
	assert.Equal(t, meta.GroupVersionKind, meta.GetGroupVersionKind())

	meta.SetName("name")
	assert.Equal(t, meta.Name, meta.GetName())

	assert.Equal(t, 0, len(meta.GetAttributes()))
	meta.SetAttributes(map[string]string{
		"abc": "123",
	})
	assert.Equal(t, meta.Attributes, meta.GetAttributes())

	assert.Equal(t, 0, len(meta.GetTags()))
	meta.SetTags([]string{"tag1", "tag2"})
	assert.Equal(t, meta.Tags, meta.GetTags())
}

// should be able to call get methods if meta is nil
func TestResourceMetaNilReference(t *testing.T) {
	var meta *ResourceMeta

	assert.Equal(t, "", meta.GetName())
	assert.Equal(t, Metadata{}, meta.GetMetadata())
	assert.Equal(t, GroupVersionKind{}, meta.GetGroupVersionKind())
	assert.Equal(t, map[string]string{}, meta.GetAttributes())
	assert.Equal(t, []string{}, meta.GetTags())
	assert.Nil(t, meta.GetSubResource("abc"))
}
