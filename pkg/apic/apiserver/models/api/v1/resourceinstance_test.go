package v1

import (
	"encoding/json"
	"io/ioutil"
	"os"
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

func TestResourceInstance_UnmarshalMarshallJSON(t *testing.T) {
	tests := []struct {
		name         string
		testFile     string
		outFile      string
		update       map[string]interface{}
		validateKeys []string
	}{
		{
			name:     "Discovery Agent",
			testFile: "testdata/discoveryagent_in.json",
			outFile:  "testdata/discoveryagent_out.json",
			update: map[string]interface{}{
				"dataplaneType": "Changed",
				"config": map[string]string{
					"test": "set",
				},
			},
			validateKeys: []string{"spec", "status", "name", "title"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputFile, _ := os.Open(tt.testFile)
			inputData, _ := ioutil.ReadAll(inputFile)

			// unmarshal the json to an object
			ri := &ResourceInstance{}
			if err := ri.UnmarshalJSON(inputData); err != nil {
				t.Errorf("ResourceInstance.UnmarshalJSON() error = %v", err)
			}

			// update the selected spec values
			for k, v := range tt.update {
				ri.Spec[k] = v
			}

			out, err := ri.MarshalJSON()
			if err != nil {
				t.Errorf("ResourceInstance.MarshalJSON() error = %v", err)
			}

			// unmarshal out to map[string]interface{} to complete compares
			outData := map[string]interface{}{}
			json.Unmarshal(out, &outData)

			// unmarshal expected to map[string]interface{}
			expectedFile, _ := os.Open(tt.outFile)
			expectedBytes, _ := ioutil.ReadAll(expectedFile)
			expectedData := map[string]interface{}{}
			json.Unmarshal(expectedBytes, &expectedData)

			// compare out and expected
			for _, k := range tt.validateKeys {
				assert.Equal(t, expectedData[k], outData[k])
			}
		})
	}
}
