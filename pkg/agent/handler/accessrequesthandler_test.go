package handler

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessRequestHandler(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		resource *v1.ResourceInstance
		action   proto.Event_Type
	}{
		{
			name:     "should save a category ResourceClient",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: accessRequest,
						},
					},
				},
			},
		},
		{
			name:     "should update a category ResourceClient",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: accessRequest,
						},
					},
				},
			},
		},
		{
			name:     "should delete a category ResourceClient",
			hasError: false,
			action:   proto.Event_DELETED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: accessRequest,
						},
					},
				},
			},
		},
		{
			name:     "should return nil when the kind is not a Category",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: category,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewAccessRequestHandler()

			err := handler.Handle(tc.action, nil, tc.resource)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}
