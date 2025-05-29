package apic

import (
	"fmt"
	"strings"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func addSpecHashToResource(h v1.Interface) error {
	ri, err := h.AsInstance()
	if err != nil {
		return err
	}

	hashInt, err := util.ComputeHash(ri.Spec)
	if err != nil {
		return err
	}

	util.SetAgentDetailsKey(h, definitions.AttrSpecHash, fmt.Sprintf("%v", hashInt))
	return nil
}

// GetSubscriptionNameFromAccessRequest - Returns the subscription name from access request references
func GetSubscriptionNameFromAccessRequest(ar *management.AccessRequest) string {
	if ar == nil {
		return ""
	}

	subscriptionName := ""
	subsRefName := getSubscriptionRefName(ar)
	if subsRefName != "" {
		refElements := strings.Split(subsRefName, "/")
		if len(refElements) == 2 && refElements[0] == "catalog" {
			subscriptionName = refElements[1]
		}
	}
	return subscriptionName
}

func getSubscriptionRefName(ar *management.AccessRequest) string {
	for _, ref := range ar.References {
		switch arRef := ref.(type) {
		case map[string]interface{}:
			kind := arRef["kind"]
			if kind == definitions.Subscription {
				return arRef["name"].(string)
			}
		case management.AccessRequestReferencesSubscription:
			return getSubscriptionName(&arRef)
		case *management.AccessRequestReferencesSubscription:
			return getSubscriptionName(arRef)
		}
	}
	return ""
}

func getSubscriptionName(arRef *management.AccessRequestReferencesSubscription) string {
	if arRef.Kind == definitions.Subscription {
		return arRef.Name
	}
	return ""
}

func addCustomFieldsToNewRequestSchema(clientLogger log.FieldLogger, data, existingRI *v1.ResourceInstance) *v1.ResourceInstance {
	// extract existing and new request schemas
	existingRequestSchema, ok := existingRI.Spec["schema"].(map[string]interface{})
	if !ok {
		clientLogger.Warn("failed to get request schema from existing ARD/CRD")
		return data
	}
	newRequestSchema, ok := data.Spec["schema"].(map[string]interface{})
	if !ok {
		clientLogger.Warn("failed to get request schema from new ARD/CRD")
		return data
	}

	// extract properties from new and existing request schemas
	existingProperties, ok := existingRequestSchema["properties"].(map[string]interface{})
	if !ok {
		clientLogger.Warn("failed to get properties from existing ARD/CRD request schema")
		return data
	}
	newProperties, ok := newRequestSchema["properties"].(map[string]interface{})
	if !ok {
		clientLogger.Warn("failed to get properties from new ARD/CRD request schema")
		return data
	}

	customFieldProps := make(map[string]interface{}, 0)
	for key, prop := range existingProperties {
		existingProp, ok := prop.(map[string]interface{})
		if !ok {
			clientLogger.Warn("failed to get property from existing ARD/CRD properties")
			return data
		}

		if _, ok := existingProp["x-custom-field"]; ok {
			customFieldProps[key] = prop
		}
	}

	// add custom field properties to new request schema properties
	for key, prop := range customFieldProps {
		newProperties[key] = prop
	}

	// update new request schema with updated properties
	newRequestSchema["properties"] = newProperties

	// update new spec with updated request schema
	data.Spec["schema"] = newRequestSchema

	return data
}
