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
	SubscriptionCtx log.ContextField = "subscription"

	_SubscriptionGVK = apiv1.GroupVersionKind{
		GroupKind: apiv1.GroupKind{
			Group: "catalog",
			Kind:  "Subscription",
		},
		APIVersion: "v1alpha1",
	}

	SubscriptionScopes = []string{""}
)

const (
	SubscriptionResourceName               = "subscriptions"
	SubscriptionApprovalSubResourceName    = "approval"
	SubscriptionBillingSubResourceName     = "billing"
	SubscriptionMarketplaceSubResourceName = "marketplace"
	SubscriptionReferencesSubResourceName  = "references"
	SubscriptionStateSubResourceName       = "state"
	SubscriptionStatusSubResourceName      = "status"
)

func SubscriptionGVK() apiv1.GroupVersionKind {
	return _SubscriptionGVK
}

func init() {
	apiv1.RegisterGVK(_SubscriptionGVK, SubscriptionScopes[0], SubscriptionResourceName)
	log.RegisterContextField(SubscriptionCtx)
}

// Subscription Resource
type Subscription struct {
	apiv1.ResourceMeta
	Approval    SubscriptionApproval    `json:"approval"`
	Billing     SubscriptionBilling     `json:"billing"`
	Marketplace SubscriptionMarketplace `json:"marketplace"`
	Owner       *apiv1.Owner            `json:"owner"`
	References  SubscriptionReferences  `json:"references"`
	Spec        SubscriptionSpec        `json:"spec"`
	State       SubscriptionState       `json:"state"`
	// Status      SubscriptionStatus      `json:"status"`
	Status *apiv1.ResourceStatus `json:"status"`
}

// NewSubscription creates an empty *Subscription
func NewSubscription(name string) *Subscription {
	return &Subscription{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             name,
			GroupVersionKind: _SubscriptionGVK,
		},
	}
}

// SubscriptionFromInstanceArray converts a []*ResourceInstance to a []*Subscription
func SubscriptionFromInstanceArray(fromArray []*apiv1.ResourceInstance) ([]*Subscription, error) {
	newArray := make([]*Subscription, 0)
	for _, item := range fromArray {
		res := &Subscription{}
		err := res.FromInstance(item)
		if err != nil {
			return make([]*Subscription, 0), err
		}
		newArray = append(newArray, res)
	}

	return newArray, nil
}

// AsInstance converts a Subscription to a ResourceInstance
func (res *Subscription) AsInstance() (*apiv1.ResourceInstance, error) {
	meta := res.ResourceMeta
	meta.GroupVersionKind = SubscriptionGVK()
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

// FromInstance converts a ResourceInstance to a Subscription
func (res *Subscription) FromInstance(ri *apiv1.ResourceInstance) error {
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
func (res *Subscription) MarshalJSON() ([]byte, error) {
	m, err := json.Marshal(&res.ResourceMeta)
	if err != nil {
		return nil, err
	}

	var out map[string]interface{}
	err = json.Unmarshal(m, &out)
	if err != nil {
		return nil, err
	}

	out["approval"] = res.Approval
	out["billing"] = res.Billing
	out["marketplace"] = res.Marketplace
	out["owner"] = res.Owner
	out["references"] = res.References
	out["spec"] = res.Spec
	out["state"] = res.State
	out["status"] = res.Status

	return json.Marshal(out)
}

// UnmarshalJSON custom unmarshaller to handle sub resources
func (res *Subscription) UnmarshalJSON(data []byte) error {
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

	// marshalling subresource Approval
	if v, ok := aux.SubResources["approval"]; ok {
		sr, err = json.Marshal(v)
		if err != nil {
			return err
		}

		delete(aux.SubResources, "approval")
		err = json.Unmarshal(sr, &res.Approval)
		if err != nil {
			return err
		}
	}

	// marshalling subresource Billing
	if v, ok := aux.SubResources["billing"]; ok {
		sr, err = json.Marshal(v)
		if err != nil {
			return err
		}

		delete(aux.SubResources, "billing")
		err = json.Unmarshal(sr, &res.Billing)
		if err != nil {
			return err
		}
	}

	// marshalling subresource Marketplace
	if v, ok := aux.SubResources["marketplace"]; ok {
		sr, err = json.Marshal(v)
		if err != nil {
			return err
		}

		delete(aux.SubResources, "marketplace")
		err = json.Unmarshal(sr, &res.Marketplace)
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

	// marshalling subresource State
	if v, ok := aux.SubResources["state"]; ok {
		sr, err = json.Marshal(v)
		if err != nil {
			return err
		}

		delete(aux.SubResources, "state")
		err = json.Unmarshal(sr, &res.State)
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
func (res *Subscription) PluralName() string {
	return SubscriptionResourceName
}
