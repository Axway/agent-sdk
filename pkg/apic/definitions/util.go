package definitions

import (
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// GetSubscriptionNameFromAccessRequest - Returns the subscription name from access request references
func GetSubscriptionNameFromAccessRequest(ar *mv1.AccessRequest) string {
	if ar == nil {
		return ""
	}

	for _, ref := range ar.References {
		switch arRef := ref.(type) {
		case map[string]interface{}:
			kind := arRef["kind"]
			if kind == "Subscription" {
				return arRef["name"].(string)
			}
		case mv1.AccessRequestReferencesSubscription:
			return getSubscriptionName(&arRef)
		case *mv1.AccessRequestReferencesSubscription:
			return getSubscriptionName(arRef)
		}
	}
	return ""
}

func getSubscriptionName(arRef *mv1.AccessRequestReferencesSubscription) string {
	if arRef.Kind == "Subscription" {
		return arRef.Name
	}
	return ""
}
