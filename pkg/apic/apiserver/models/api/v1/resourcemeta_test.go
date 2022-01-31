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

	assert.Equal(t, meta1.Name, meta2.Name)
	assert.Equal(t, meta1.Title, meta2.Title)
	assert.Equal(t, meta1.Metadata.ID, meta2.Metadata.ID)
	assert.Equal(t, meta1.SubResources, meta2.SubResources)
	assert.Equal(t, meta1.Tags, meta2.Tags)
	assert.Equal(t, meta1.Attributes, meta2.Attributes)

	// expect to only find the x-agent-details key on the MetaResource.SubResource field
	assert.True(t, len(meta2.SubResources) == 1)
	assert.NotEmpty(t, meta2.SubResources["x-agent-details"])
}
