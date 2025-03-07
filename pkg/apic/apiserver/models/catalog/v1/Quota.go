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
	QuotaCtx log.ContextField = "quota"

	_QuotaGVK = apiv1.GroupVersionKind{
		GroupKind: apiv1.GroupKind{
			Group: "catalog",
			Kind:  "Quota",
		},
		APIVersion: "v1",
	}

	QuotaScopes = []string{"ProductPlan"}
)

const (
	QuotaResourceName          = "quotas"
	QuotaStatusSubResourceName = "status"
)

func QuotaGVK() apiv1.GroupVersionKind {
	return _QuotaGVK
}

func init() {
	apiv1.RegisterGVK(_QuotaGVK, QuotaScopes[0], QuotaResourceName)
	log.RegisterContextField(QuotaCtx)
}

// Quota Resource
type Quota struct {
	apiv1.ResourceMeta
	Owner *apiv1.Owner `json:"owner"`
	Spec  QuotaSpec    `json:"spec"`
	// Status QuotaStatus  `json:"status"`
	Status *apiv1.ResourceStatus `json:"status"`
}

// NewQuota creates an empty *Quota
func NewQuota(name, scopeName string) *Quota {
	return &Quota{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             name,
			GroupVersionKind: _QuotaGVK,
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Name: scopeName,
					Kind: QuotaScopes[0],
				},
			},
		},
	}
}

// QuotaFromInstanceArray converts a []*ResourceInstance to a []*Quota
func QuotaFromInstanceArray(fromArray []*apiv1.ResourceInstance) ([]*Quota, error) {
	newArray := make([]*Quota, 0)
	for _, item := range fromArray {
		res := &Quota{}
		err := res.FromInstance(item)
		if err != nil {
			return make([]*Quota, 0), err
		}
		newArray = append(newArray, res)
	}

	return newArray, nil
}

// AsInstance converts a Quota to a ResourceInstance
func (res *Quota) AsInstance() (*apiv1.ResourceInstance, error) {
	meta := res.ResourceMeta
	meta.GroupVersionKind = QuotaGVK()
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

// FromInstance converts a ResourceInstance to a Quota
func (res *Quota) FromInstance(ri *apiv1.ResourceInstance) error {
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
func (res *Quota) MarshalJSON() ([]byte, error) {
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
	out["status"] = res.Status

	return json.Marshal(out)
}

// UnmarshalJSON custom unmarshaller to handle sub resources
func (res *Quota) UnmarshalJSON(data []byte) error {
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
func (res *Quota) PluralName() string {
	return QuotaResourceName
}
