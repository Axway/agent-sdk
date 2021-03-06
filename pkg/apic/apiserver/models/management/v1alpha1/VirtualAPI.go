/*
 * This file is automatically generated
 */

package v1alpha1

import (
	"encoding/json"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

var (
	_VirtualAPIGVK = apiv1.GroupVersionKind{
		GroupKind: apiv1.GroupKind{
			Group: "management",
			Kind:  "VirtualAPI",
		},
		APIVersion: "v1alpha1",
	}
)

const (
	VirtualAPIScope = ""

	VirtualAPIResourceName = "virtualapis"
)

func VirtualAPIGVK() apiv1.GroupVersionKind {
	return _VirtualAPIGVK
}

func init() {
	apiv1.RegisterGVK(_VirtualAPIGVK, VirtualAPIScope, VirtualAPIResourceName)
}

// VirtualAPI Resource
type VirtualAPI struct {
	apiv1.ResourceMeta

	Icon interface{} `json:"icon"`

	Owner interface{} `json:"owner"`

	Spec VirtualApiSpec `json:"spec"`

	State interface{} `json:"state"`
}

// FromInstance converts a ResourceInstance to a VirtualAPI
func (res *VirtualAPI) FromInstance(ri *apiv1.ResourceInstance) error {
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

// VirtualAPIFromInstanceArray converts a []*ResourceInstance to a []*VirtualAPI
func VirtualAPIFromInstanceArray(fromArray []*apiv1.ResourceInstance) ([]*VirtualAPI, error) {
	newArray := make([]*VirtualAPI, 0)
	for _, item := range fromArray {
		res := &VirtualAPI{}
		err := res.FromInstance(item)
		if err != nil {
			return make([]*VirtualAPI, 0), err
		}
		newArray = append(newArray, res)
	}

	return newArray, nil
}

// AsInstance converts a VirtualAPI to a ResourceInstance
func (res *VirtualAPI) AsInstance() (*apiv1.ResourceInstance, error) {
	meta := res.ResourceMeta
	meta.GroupVersionKind = VirtualAPIGVK()
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
