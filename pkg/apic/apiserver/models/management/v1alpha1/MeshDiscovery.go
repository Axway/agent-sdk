/*
 * This file is automatically generated
 */

package v1alpha1

import (
	"encoding/json"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

var (
	_MeshDiscoveryGVK = apiv1.GroupVersionKind{
		GroupKind: apiv1.GroupKind{
			Group: "management",
			Kind:  "MeshDiscovery",
		},
		APIVersion: "v1alpha1",
	}
)

const (
	MeshDiscoveryScope = "Mesh"

	MeshDiscoveryResourceName = "meshdiscoveries"
)

func MeshDiscoveryGVK() apiv1.GroupVersionKind {
	return _MeshDiscoveryGVK
}

func init() {
	apiv1.RegisterGVK(_MeshDiscoveryGVK, MeshDiscoveryScope, MeshDiscoveryResourceName)
}

// MeshDiscovery Resource
type MeshDiscovery struct {
	apiv1.ResourceMeta

	Owner interface{} `json:"owner"`

	Spec MeshDiscoverySpec `json:"spec"`
}

// FromInstance converts a ResourceInstance to a MeshDiscovery
func (res *MeshDiscovery) FromInstance(ri *apiv1.ResourceInstance) error {
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

// MeshDiscoveryFromInstanceArray converts a []*ResourceInstance to a []*MeshDiscovery
func MeshDiscoveryFromInstanceArray(fromArray []*apiv1.ResourceInstance) ([]*MeshDiscovery, error) {
	newArray := make([]*MeshDiscovery, 0)
	for _, item := range fromArray {
		res := &MeshDiscovery{}
		err := res.FromInstance(item)
		if err != nil {
			return make([]*MeshDiscovery, 0), err
		}
		newArray = append(newArray, res)
	}

	return newArray, nil
}

// AsInstance converts a MeshDiscovery to a ResourceInstance
func (res *MeshDiscovery) AsInstance() (*apiv1.ResourceInstance, error) {
	meta := res.ResourceMeta
	meta.GroupVersionKind = MeshDiscoveryGVK()
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
