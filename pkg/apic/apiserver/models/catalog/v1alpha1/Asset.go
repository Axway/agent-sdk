/*
 * This file is automatically generated
 */

package v1alpha1

import (
	"encoding/json"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

var (
	_AssetGVK = apiv1.GroupVersionKind{
		GroupKind: apiv1.GroupKind{
			Group: "catalog",
			Kind:  "Asset",
		},
		APIVersion: "v1alpha1",
	}
)

const (
	AssetScope = ""

	AssetResourceName = "assets"
)

func AssetGVK() apiv1.GroupVersionKind {
	return _AssetGVK
}

func init() {
	apiv1.RegisterGVK(_AssetGVK, AssetScope, AssetResourceName)
}

// Asset Resource
type Asset struct {
	apiv1.ResourceMeta

	Icon interface{} `json:"icon"`

	Owner interface{} `json:"owner"`

	References AssetReferences `json:"references"`

	Spec AssetSpec `json:"spec"`

	State interface{} `json:"state"`
}

// FromInstance converts a ResourceInstance to a Asset
func (res *Asset) FromInstance(ri *apiv1.ResourceInstance) error {
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
	return err
}

// AssetFromInstanceArray converts a []*ResourceInstance to a []*Asset
func AssetFromInstanceArray(fromArray []*apiv1.ResourceInstance) ([]*Asset, error) {
	newArray := make([]*Asset, 0)
	for _, item := range fromArray {
		res := &Asset{}
		err := res.FromInstance(item)
		if err != nil {
			return make([]*Asset, 0), err
		}
		newArray = append(newArray, res)
	}

	return newArray, nil
}

// AsInstance converts a Asset to a ResourceInstance
func (res *Asset) AsInstance() (*apiv1.ResourceInstance, error) {
	meta := res.ResourceMeta
	meta.GroupVersionKind = AssetGVK()
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

	return &instance, nil
}
