package v1alpha1

import (
	"encoding/json"
	"testing"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestAPIServiceMarshal(t *testing.T) {
	svc := &APIService{
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
		Spec: ApiServiceSpec{
			Description: "desc",
			Categories:  []string{"cat1", "cat2"},
			Icon: ApiServiceSpecIcon{
				ContentType: "image/png",
				Data:        "data",
			},
		},
	}

	bts, err := json.Marshal(svc)
	assert.Nil(t, err)
	assert.NotNil(t, bts)

	svc2 := &APIService{}

	err = json.Unmarshal(bts, svc2)
	assert.Nil(t, err)
}
