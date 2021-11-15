package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/apic"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/cache"
)

func TestNewAPISvcHandler(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		resource *apiv1.ResourceInstance
		action   proto.Event_Type
	}{
		{
			name:     "should save a resource that has an externalAPIID attribute, and no externalAPIPrimaryKey attribute",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: APIService,
						},
					},
					Attributes: map[string]string{
						apic.AttrExternalAPIID:   "123",
						apic.AttrExternalAPIName: "name",
					},
				},
			},
		},
		{
			name:     "should save a resource that has an externalAPIID attribute, and has the externalAPIPrimaryKey attribute",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: APIService,
						},
					},
					Attributes: map[string]string{
						apic.AttrExternalAPIID:         "123",
						apic.AttrExternalAPIPrimaryKey: "abc",
						apic.AttrExternalAPIName:       "name",
					},
				},
			},
		},
		{
			name:     "should fail to save the item to the cache when the externalAPIID attribute is not found",
			hasError: true,
			action:   proto.Event_CREATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: APIService,
						},
					},
					Attributes: map[string]string{},
				},
			},
		},
		{
			name:     "should handle a delete action",
			hasError: false,
			action:   proto.Event_DELETED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: APIService,
						},
					},
					Attributes: map[string]string{
						apic.AttrExternalAPIID:   "123",
						apic.AttrExternalAPIName: "name",
					},
				},
			},
		},
		{
			name:     "should return nil when the resource kind is not an APIService",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: Category,
						},
					},
					Attributes: map[string]string{},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewAPISvcHandler(&mockCache{})

			err := handler.callback(tc.action, tc.resource)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}

type mockCache struct {
}

func (m mockCache) Get(_ string) (interface{}, error) {
	return nil, nil
}

func (m mockCache) GetItem(_ string) (*cache.Item, error) {
	return nil, nil
}

func (m mockCache) GetBySecondaryKey(_ string) (interface{}, error) {
	return nil, nil
}

func (m mockCache) GetItemBySecondaryKey(_ string) (*cache.Item, error) {
	return nil, nil
}

func (m mockCache) GetForeignKeys() []string {
	return nil
}

func (m mockCache) GetItemsByForeignKey(_ string) ([]*cache.Item, error) {
	return nil, nil
}

func (m mockCache) GetKeys() []string {
	return nil
}

func (m mockCache) HasItemChanged(_ string, _ interface{}) (bool, error) {
	return false, nil
}

func (m mockCache) HasItemBySecondaryKeyChanged(_ string, _ interface{}) (bool, error) {
	return false, nil
}

func (m mockCache) Set(_ string, _ interface{}) error {
	return nil
}

func (m mockCache) SetWithSecondaryKey(_ string, _ string, _ interface{}) error {
	return nil
}

func (m mockCache) SetWithForeignKey(_ string, _ string, _ interface{}) error {
	return nil
}

func (m mockCache) SetSecondaryKey(_ string, _ string) error {
	return nil
}

func (m mockCache) SetForeignKey(_ string, _ string) error {
	return nil
}

func (m mockCache) Delete(_ string) error {
	return nil
}

func (m mockCache) DeleteBySecondaryKey(_ string) error {
	return nil
}

func (m mockCache) DeleteSecondaryKey(_ string) error {
	return nil
}

func (m mockCache) DeleteForeignKey(_ string) error {
	return nil
}

func (m mockCache) DeleteItemsByForeignKey(_ string) error {
	return nil
}

func (m mockCache) Flush() {
}

func (m mockCache) Save(_ string) error {
	return nil
}

func (m mockCache) Load(_ string) error {
	return nil
}
