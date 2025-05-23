/*
 * This file is automatically generated
 */

package catalog

import (
	"encoding/json"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

var (
	AssetReleaseCtx log.ContextField = "assetRelease"

	_AssetReleaseGVK = apiv1.GroupVersionKind{
		GroupKind: apiv1.GroupKind{
			Group: "catalog",
			Kind:  "AssetRelease",
		},
		APIVersion: "v1alpha1",
	}

	AssetReleaseScopes = []string{""}
)

const (
	AssetReleaseResourceName              = "assetreleases"
	AssetReleaseIconSubResourceName       = "icon"
	AssetReleaseReferencesSubResourceName = "references"
	AssetReleaseStatusSubResourceName     = "status"
)

func AssetReleaseGVK() apiv1.GroupVersionKind {
	return _AssetReleaseGVK
}

func init() {
	apiv1.RegisterGVK(_AssetReleaseGVK, AssetReleaseScopes[0], AssetReleaseResourceName)
	log.RegisterContextField(AssetReleaseCtx)
}

// AssetRelease Resource
type AssetRelease struct {
	apiv1.ResourceMeta
	Icon       interface{}      `json:"icon"`
	Owner      *apiv1.Owner     `json:"owner"`
	References interface{}      `json:"references"`
	Spec       AssetReleaseSpec `json:"spec"`
	// Status     AssetReleaseStatus `json:"status"`
	Status *apiv1.ResourceStatus `json:"status"`
}

// NewAssetRelease creates an empty *AssetRelease
func NewAssetRelease(name string) *AssetRelease {
	return &AssetRelease{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             name,
			GroupVersionKind: _AssetReleaseGVK,
		},
	}
}

// AssetReleaseFromInstanceArray converts a []*ResourceInstance to a []*AssetRelease
func AssetReleaseFromInstanceArray(fromArray []*apiv1.ResourceInstance) ([]*AssetRelease, error) {
	newArray := make([]*AssetRelease, 0)
	for _, item := range fromArray {
		res := &AssetRelease{}
		err := res.FromInstance(item)
		if err != nil {
			return make([]*AssetRelease, 0), err
		}
		newArray = append(newArray, res)
	}

	return newArray, nil
}

// AsInstance converts a AssetRelease to a ResourceInstance
func (res *AssetRelease) AsInstance() (*apiv1.ResourceInstance, error) {
	meta := res.ResourceMeta
	meta.GroupVersionKind = AssetReleaseGVK()
	res.ResourceMeta = meta

	m, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	instance := apiv1.ResourceInstance{}
	err = json.Unmarshal(m, &instance)
	if err != nil {
		return nil, err
	}
	instance.SubResourceHashes = res.SubResourceHashes
	return &instance, nil
}

// FromInstance converts a ResourceInstance to a AssetRelease
func (res *AssetRelease) FromInstance(ri *apiv1.ResourceInstance) error {
	if ri == nil {
		res = nil
		return nil
	}
	var err error
	rawResource := ri.GetRawResource()
	if rawResource == nil {
		rawResource, err = json.Marshal(ri)
		if err != nil {
			return err
		}
	}
	err = json.Unmarshal(rawResource, res)
	if err != nil {
		return err
	}
	res.SubResourceHashes = ri.SubResourceHashes
	return err
}

// MarshalJSON custom marshaller to handle sub resources
func (res *AssetRelease) MarshalJSON() ([]byte, error) {
	m, err := json.Marshal(&res.ResourceMeta)
	if err != nil {
		return nil, err
	}

	var out map[string]interface{}
	err = json.Unmarshal(m, &out)
	if err != nil {
		return nil, err
	}

	out["icon"] = res.Icon
	out["owner"] = res.Owner
	out["references"] = res.References
	out["spec"] = res.Spec
	out["status"] = res.Status

	return json.Marshal(out)
}

// UnmarshalJSON custom unmarshaller to handle sub resources
func (res *AssetRelease) UnmarshalJSON(data []byte) error {
	var err error

	aux := &apiv1.ResourceInstance{}
	err = json.Unmarshal(data, aux)
	if err != nil {
		return err
	}

	res.ResourceMeta = aux.ResourceMeta
	res.Owner = aux.Owner

	// ResourceInstance holds the spec as a map[string]interface{}.
	// Convert it to bytes, then convert to the spec type for the resource.
	sr, err := json.Marshal(aux.Spec)
	if err != nil {
		return err
	}

	err = json.Unmarshal(sr, &res.Spec)
	if err != nil {
		return err
	}

	// marshalling subresource Icon
	if v, ok := aux.SubResources["icon"]; ok {
		sr, err = json.Marshal(v)
		if err != nil {
			return err
		}

		delete(aux.SubResources, "icon")
		err = json.Unmarshal(sr, &res.Icon)
		if err != nil {
			return err
		}
	}

	// marshalling subresource References
	if v, ok := aux.SubResources["references"]; ok {
		sr, err = json.Marshal(v)
		if err != nil {
			return err
		}

		delete(aux.SubResources, "references")
		err = json.Unmarshal(sr, &res.References)
		if err != nil {
			return err
		}
	}

	// marshalling subresource Status
	if v, ok := aux.SubResources["status"]; ok {
		sr, err = json.Marshal(v)
		if err != nil {
			return err
		}

		delete(aux.SubResources, "status")
		// err = json.Unmarshal(sr, &res.Status)
		res.Status = &apiv1.ResourceStatus{}
		err = json.Unmarshal(sr, res.Status)
		if err != nil {
			return err
		}
	}

	return nil
}

// PluralName returns the plural name of the resource
func (res *AssetRelease) PluralName() string {
	return AssetReleaseResourceName
}
