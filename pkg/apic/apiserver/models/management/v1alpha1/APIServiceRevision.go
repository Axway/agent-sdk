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
	APIServiceRevisionCtx log.ContextField = "apiServiceRevision"

	_APIServiceRevisionGVK = apiv1.GroupVersionKind{
		GroupKind: apiv1.GroupKind{
			Group: "management",
			Kind:  "APIServiceRevision",
		},
		APIVersion: "v1alpha1",
	}

	APIServiceRevisionScopes = []string{"Environment"}
)

const (
	APIServiceRevisionResourceName              = "apiservicerevisions"
	ApiServiceRevisionComplianceSubResourceName = "compliance"
)

func APIServiceRevisionGVK() apiv1.GroupVersionKind {
	return _APIServiceRevisionGVK
}

func init() {
	apiv1.RegisterGVK(_APIServiceRevisionGVK, APIServiceRevisionScopes[0], APIServiceRevisionResourceName)
	log.RegisterContextField(APIServiceRevisionCtx)
}

// APIServiceRevision Resource
type APIServiceRevision struct {
	apiv1.ResourceMeta
	Compliance ApiServiceRevisionCompliance `json:"compliance"`
	Owner      *apiv1.Owner                 `json:"owner"`
	Spec       ApiServiceRevisionSpec       `json:"spec"`
}

// NewAPIServiceRevision creates an empty *APIServiceRevision
func NewAPIServiceRevision(name, scopeName string) *APIServiceRevision {
	return &APIServiceRevision{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             name,
			GroupVersionKind: _APIServiceRevisionGVK,
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Name: scopeName,
					Kind: APIServiceRevisionScopes[0],
				},
			},
		},
	}
}

// APIServiceRevisionFromInstanceArray converts a []*ResourceInstance to a []*APIServiceRevision
func APIServiceRevisionFromInstanceArray(fromArray []*apiv1.ResourceInstance) ([]*APIServiceRevision, error) {
	newArray := make([]*APIServiceRevision, 0)
	for _, item := range fromArray {
		res := &APIServiceRevision{}
		err := res.FromInstance(item)
		if err != nil {
			return make([]*APIServiceRevision, 0), err
		}
		newArray = append(newArray, res)
	}

	return newArray, nil
}

// AsInstance converts a APIServiceRevision to a ResourceInstance
func (res *APIServiceRevision) AsInstance() (*apiv1.ResourceInstance, error) {
	meta := res.ResourceMeta
	meta.GroupVersionKind = APIServiceRevisionGVK()
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

// FromInstance converts a ResourceInstance to a APIServiceRevision
func (res *APIServiceRevision) FromInstance(ri *apiv1.ResourceInstance) error {
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
func (res *APIServiceRevision) MarshalJSON() ([]byte, error) {
	m, err := json.Marshal(&res.ResourceMeta)
	if err != nil {
		return nil, err
	}

	var out map[string]interface{}
	err = json.Unmarshal(m, &out)
	if err != nil {
		return nil, err
	}

	out["compliance"] = res.Compliance
	out["owner"] = res.Owner
	out["spec"] = res.Spec

	return json.Marshal(out)
}

// UnmarshalJSON custom unmarshaller to handle sub resources
func (res *APIServiceRevision) UnmarshalJSON(data []byte) error {
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

	// marshalling subresource Compliance
	if v, ok := aux.SubResources["compliance"]; ok {
		sr, err = json.Marshal(v)
		if err != nil {
			return err
		}

		delete(aux.SubResources, "compliance")
		err = json.Unmarshal(sr, &res.Compliance)
		if err != nil {
			return err
		}
	}

	return nil
}

// PluralName returns the plural name of the resource
func (res *APIServiceRevision) PluralName() string {
	return APIServiceRevisionResourceName
}
