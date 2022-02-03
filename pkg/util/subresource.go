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

// GetAgentDetailsValue get a single string value fom the x-agent-details sub resource.
// Returns nil if x-agent-details does not exist.
// returns errors if unable to perform type conversion.
func GetAgentDetailsValue(h handler, key string) (string, error) {
	item := h.GetSubResource(definitions.XAgentDetails)
	if item == nil {
		return "", nil
	}

	sub, ok := item.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unable to convert x-agent-details to map[string]interface{}")
	}

	v, ok := sub[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in x-agent-details", key)
	}

	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("value for %s is not a string. found %s", key, v)
	}

	return s, nil
}

// SetAgentDetailsKey sets a key value pair in the x-agent-details sub resource
func SetAgentDetailsKey(h handler, key string, value interface{}) error {
	item := h.GetSubResource(definitions.XAgentDetails)
	if item == nil {
		h.SetSubResource(definitions.XAgentDetails, map[string]interface{}{
			key: value,
		})
		return nil
	}

	sub, ok := item.(map[string]interface{})
	if !ok {
		return fmt.Errorf("x-agent-details is not a map[string]interface{}")
	}
	sub[key] = value

	h.SetSubResource(definitions.XAgentDetails, sub)
	return nil
}
