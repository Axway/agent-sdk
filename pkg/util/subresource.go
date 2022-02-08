package util

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"
)

// handler is an interface for handling sub resources on apiserver items
type handler interface {
	// GetSubResource get a sub resource by name
	GetSubResource(key string) interface{}
	// SetSubResource saves a value to a sub resource by name and overrides the current value.
	SetSubResource(key string, resource interface{})
}

// GetAgentDetails get all the values for the x-agent-details sub resource
func GetAgentDetails(h handler) map[string]interface{} {
	item := h.GetSubResource(definitions.XAgentDetails)
	if item == nil {
		return nil
	}

	v, ok := item.(map[string]interface{})
	if !ok {
		return nil
	}

	return v
}

// GetAgentDetailsValue gets a single string value fom the x-agent-details sub resource.
// Returns nil if x-agent-details does not exist.
// Returns errors if unable to perform type conversion.
// Returns an empty string if the value does not exist, or if there is an error.
func GetAgentDetailsValue(h handler, key string) (string, error) {
	item := h.GetSubResource(definitions.XAgentDetails)
	if item == nil {
		return "", nil
	}

	sub, ok := item.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf(
			"unable to convert %s to map[string]interface{}. Received type %T",
			definitions.XAgentDetails,
			sub,
		)
	}

	item, ok = sub[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in %s", key, definitions.XAgentDetails)
	}

	switch v := item.(type) {
	case int:
		return fmt.Sprintf("%d", v), nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf(
			"%s keys should be a string or int. Received type %T for key %s",
			definitions.XAgentDetails,
			v,
			key,
		)
	}
}

// SetAgentDetailsKey sets a key value pair in the x-agent-details sub resource. If x-agent-details does not exist, it is created.
// If value is not a string or an int, an error will be returned.
func SetAgentDetailsKey(h handler, key, value string) error {
	item := h.GetSubResource(definitions.XAgentDetails)
	if item == nil {
		h.SetSubResource(definitions.XAgentDetails, map[string]interface{}{key: value})
		return nil
	}

	sub, ok := item.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s is not a map[string]interface{}. Received type %T", definitions.XAgentDetails, sub)
	}

	sub[key] = value

	h.SetSubResource(definitions.XAgentDetails, sub)
	return nil
}
