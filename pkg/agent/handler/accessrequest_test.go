package handler

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
			name:     "should handle a create event for an AccessRequest when status is pending, and state is provision",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:             "name",
					Title:            "title",
					GroupVersionKind: mv1.AccessRequestGVK(),
				},
			},
		},
		{
			name:     "should handle an update event for an AccessRequest when status is pending, and state is provision",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:             "name",
					Title:            "title",
					GroupVersionKind: mv1.AccessRequestGVK(),
				},
			},
		},
		{
			name:     "should handle an update event for an AccessRequest when status is pending, and state is deprovision",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:             "name",
					Title:            "title",
					GroupVersionKind: mv1.AccessRequestGVK(),
				},
			},
		},
		{
			name:     "should deprovision when receiving a delete event",
			hasError: false,
			action:   proto.Event_DELETED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:             "name",
					Title:            "title",
					GroupVersionKind: mv1.AccessRequestGVK(),
				},
			},
		},
		{
			name:     "should return nil when the kind is not a AccessRequest",
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
