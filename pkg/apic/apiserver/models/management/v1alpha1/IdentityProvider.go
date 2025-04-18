/*
 * This file is automatically generated
 */

package management

import (
	"encoding/json"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

var (
	IdentityProviderCtx log.ContextField = "identityProvider"

	_IdentityProviderGVK = apiv1.GroupVersionKind{
		GroupKind: apiv1.GroupKind{
			Group: "management",
			Kind:  "IdentityProvider",
		},
		APIVersion: "v1alpha1",
	}

	IdentityProviderScopes = []string{""}
)

const (
	IdentityProviderResourceName            = "identityproviders"
	IdentityProviderSecuritySubResourceName = "security"
	IdentityProviderStatusSubResourceName   = "status"
)

func IdentityProviderGVK() apiv1.GroupVersionKind {
	return _IdentityProviderGVK
}

func init() {
	apiv1.RegisterGVK(_IdentityProviderGVK, IdentityProviderScopes[0], IdentityProviderResourceName)
	log.RegisterContextField(IdentityProviderCtx)
}

// IdentityProvider Resource
type IdentityProvider struct {
	apiv1.ResourceMeta
	Owner    *apiv1.Owner             `json:"owner"`
	Security IdentityProviderSecurity `json:"security"`
	Spec     IdentityProviderSpec     `json:"spec"`
	// Status   IdentityProviderStatus   `json:"status"`
	Status *apiv1.ResourceStatus `json:"status"`
}

// NewIdentityProvider creates an empty *IdentityProvider
func NewIdentityProvider(name string) *IdentityProvider {
	return &IdentityProvider{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             name,
			GroupVersionKind: _IdentityProviderGVK,
		},
	}
}

// IdentityProviderFromInstanceArray converts a []*ResourceInstance to a []*IdentityProvider
func IdentityProviderFromInstanceArray(fromArray []*apiv1.ResourceInstance) ([]*IdentityProvider, error) {
	newArray := make([]*IdentityProvider, 0)
	for _, item := range fromArray {
		res := &IdentityProvider{}
		err := res.FromInstance(item)
		if err != nil {
			return make([]*IdentityProvider, 0), err
		}
		newArray = append(newArray, res)
	}

	return newArray, nil
}

// AsInstance converts a IdentityProvider to a ResourceInstance
func (res *IdentityProvider) AsInstance() (*apiv1.ResourceInstance, error) {
	meta := res.ResourceMeta
	meta.GroupVersionKind = IdentityProviderGVK()
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

// FromInstance converts a ResourceInstance to a IdentityProvider
func (res *IdentityProvider) FromInstance(ri *apiv1.ResourceInstance) error {
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
func (res *IdentityProvider) MarshalJSON() ([]byte, error) {
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
	out["security"] = res.Security
	out["spec"] = res.Spec
	out["status"] = res.Status

	return json.Marshal(out)
}

// UnmarshalJSON custom unmarshaller to handle sub resources
func (res *IdentityProvider) UnmarshalJSON(data []byte) error {
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

	// marshalling subresource Security
	if v, ok := aux.SubResources["security"]; ok {
		sr, err = json.Marshal(v)
		if err != nil {
			return err
		}

		delete(aux.SubResources, "security")
		err = json.Unmarshal(sr, &res.Security)
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
func (res *IdentityProvider) PluralName() string {
	return IdentityProviderResourceName
}
