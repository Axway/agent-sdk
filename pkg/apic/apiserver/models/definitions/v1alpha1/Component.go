/*
 * This file is automatically generated
 */

package definitions

import (
	"encoding/json"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

var (
	ComponentCtx log.ContextField = "component"

	_ComponentGVK = apiv1.GroupVersionKind{
		GroupKind: apiv1.GroupKind{
			Group: "definitions",
			Kind:  "Component",
		},
		APIVersion: "v1alpha1",
	}

	ComponentScopes = []string{""}
)

const (
	ComponentResourceName = "components"
)

func ComponentGVK() apiv1.GroupVersionKind {
	return _ComponentGVK
}

func init() {
	apiv1.RegisterGVK(_ComponentGVK, ComponentScopes[0], ComponentResourceName)
	log.RegisterContextField(ComponentCtx)
}

// Component Resource
type Component struct {
	apiv1.ResourceMeta
	Owner *apiv1.Owner  `json:"owner"`
	Spec  ComponentSpec `json:"spec"`
}

// NewComponent creates an empty *Component
func NewComponent(name string) *Component {
	return &Component{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             name,
			GroupVersionKind: _ComponentGVK,
		},
	}
}

// ComponentFromInstanceArray converts a []*ResourceInstance to a []*Component
func ComponentFromInstanceArray(fromArray []*apiv1.ResourceInstance) ([]*Component, error) {
	newArray := make([]*Component, 0)
	for _, item := range fromArray {
		res := &Component{}
		err := res.FromInstance(item)
		if err != nil {
			return make([]*Component, 0), err
		}
		newArray = append(newArray, res)
	}

	return newArray, nil
}

// AsInstance converts a Component to a ResourceInstance
func (res *Component) AsInstance() (*apiv1.ResourceInstance, error) {
	meta := res.ResourceMeta
	meta.GroupVersionKind = ComponentGVK()
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

// FromInstance converts a ResourceInstance to a Component
func (res *Component) FromInstance(ri *apiv1.ResourceInstance) error {
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

// MarshalJSON custom marshaller to handle sub resources
func (res *Component) MarshalJSON() ([]byte, error) {
	m, err := json.Marshal(&res.ResourceMeta)
	if err != nil {
		return nil, err
	}

	var out map[string]interface{}
	err = json.Unmarshal(m, &out)
	if err != nil {
		return nil, err
	}

	out["owner"] = res.Owner
	out["spec"] = res.Spec

	return json.Marshal(out)
}

// UnmarshalJSON custom unmarshaller to handle sub resources
func (res *Component) UnmarshalJSON(data []byte) error {
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

	return nil
}

// PluralName returns the plural name of the resource
func (res *Component) PluralName() string {
	return ComponentResourceName
}