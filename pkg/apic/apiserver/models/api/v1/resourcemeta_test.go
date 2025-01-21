package v1

import (
	"encoding/json"
	"testing"

	"github.com/Axway/agent-sdk/pkg/util"
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
	assert.Equal(t, len(meta2.SubResources), len(meta1.SubResources))
	assert.Equal(t, xAgentDetailsSub, meta2.GetSubResource("x-agent-details"))

	// unset the name
	meta1.Name = ""

	bts, err = json.Marshal(meta1)
	assert.Nil(t, err)
	assert.NotNil(t, bts)

	// Unmarshal the ResourceMeta bytes to a map to confirm that MarshalJSON did not save name
	values = map[string]interface{}{}
	json.Unmarshal(bts, &values)
	assert.NotContains(t, values, "name")

	// Unmarshal the ResourceMeta bytes to a map to confirm that MarshalJSON
	meta3 := ResourceMeta{}
	err = json.Unmarshal(bts, &meta3)
	assert.Nil(t, err)

	assert.Equal(t, "", meta3.Name)
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
			References: []Reference{
				{
					Group:     "ref1group",
					Kind:      "ref1kind",
					ID:        "ref1id",
					Name:      "ref1name",
					ScopeName: "ref1scope",
					Type:      "ref1type",
				},
				{
					Group:     "ref2group",
					Kind:      "ref2kind",
					ID:        "ref2id",
					Name:      "ref2name",
					ScopeName: "ref2scope",
					Type:      "ref2type",
				},
				{
					Group:     "ref3group",
					Kind:      "ref3kind",
					ID:        "ref3id",
					Name:      "ref3name",
					ScopeName: "ref3scope",
					Type:      "ref3type",
				},
			},
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

	// test GetReferenceByGVK
	ref1GVK := GroupVersionKind{
		GroupKind: GroupKind{
			Group: "ref1group",
			Kind:  "ref1kind",
		},
		APIVersion: "v1",
	}
	ref2GVK := GroupVersionKind{
		GroupKind: GroupKind{
			Group: "ref2group",
			Kind:  "ref2kind",
		},
		APIVersion: "v1",
	}
	ref3GVK := GroupVersionKind{
		GroupKind: GroupKind{
			Group: "ref3group",
			Kind:  "ref3kind",
		},
		APIVersion: "v1",
	}
	ref1test := meta.GetReferenceByGVK(ref1GVK)
	assert.Equal(t, ref1test, meta.Metadata.References[0])
	ref2test := meta.GetReferenceByGVK(ref2GVK)
	assert.Equal(t, ref2test, meta.Metadata.References[1])
	ref3test := meta.GetReferenceByGVK(ref3GVK)
	assert.Equal(t, ref3test, meta.Metadata.References[2])
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

func TestResourceMetaHashes(t *testing.T) {
	meta1 := &ResourceMeta{
		GroupVersionKind: GroupVersionKind{
			GroupKind: GroupKind{
				Group: "group",
				Kind:  "kind",
			},
		},
		Title: "title",
		Metadata: Metadata{
			ID: "333",
		},
		Name: "name",
	}
	meta1.SetSubResource("sub1", "sth")
	meta1.SetSubResource("sub2", "sth")

	bts, err := json.Marshal(meta1)
	assert.Nil(t, err)

	meta2 := &ResourceMeta{}
	err = json.Unmarshal(bts, meta2)
	assert.Nil(t, err)

	// Test that after marshal-unmarshal, the data is the same.
	meta1.Metadata.Audit = AuditMetadata{}
	meta2.Metadata.Audit = AuditMetadata{}
	assert.Equal(t, meta1, meta2)

	meta1.PrepareHashesForSending()
	meta1.SetIncomingHashes()
	meta2.CreateHashes()
	assert.Equal(t, meta1, meta2)

	hashVal, ok := meta1.GetSubResourceHash("sub1")
	assert.True(t, ok)
	hashedReal, err := util.ComputeHash("sth")
	assert.Nil(t, err)
	assert.Equal(t, hashVal, float64(hashedReal))

	meta3 := &ResourceMeta{}
	meta3.ClearHashes()
}

func TestResourceMetaGetSelfLink(t *testing.T) {
	plurals["kind"] = "kinds"
	plurals["scopeKind"] = "scopeKinds"

	meta := &ResourceMeta{
		GroupVersionKind: GroupVersionKind{
			GroupKind: GroupKind{
				Group: "group",
				Kind:  "kind",
			},
		},
		Title: "title",
		Metadata: Metadata{
			ID: "333",
		},
		Name: "name",
	}

	// no version
	link := meta.GetSelfLink()
	assert.Equal(t, "", link)

	meta.APIVersion = "v1"

	// no scope
	link = meta.GetSelfLink()
	assert.Equal(t, "/group/v1/kinds/name", link)

	meta.Metadata.Scope = MetadataScope{
		Name: "scope",
		Kind: "scopeKind",
	}

	scopeKindMap[meta.GroupKind] = "scopeKind"

	// no scope kind
	link = meta.GetSelfLink()
	assert.Equal(t, "/group/v1/scopeKinds/scope/kinds/name", link)

	// selflink
	meta.Metadata.SelfLink = "/selflink"
	link = meta.GetSelfLink()
	assert.Equal(t, "/selflink", link)
}
