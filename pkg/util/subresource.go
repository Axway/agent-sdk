package util

import (
	"fmt"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
)

// handler is an interface for handling sub resources on apiserver items
type handler interface {
	// GetSubResource gets a sub resource by name
	GetSubResource(key string) interface{}
	// SetSubResource saves a value to a sub resource by name and overrides the current value.
	SetSubResource(key string, resource interface{})
}

// GetAgentDetails get all the values for the x-agent-details sub resource
func GetAgentDetails(h handler) map[string]interface{} {
	if h == nil {
		return nil
	}

	item := h.GetSubResource(defs.XAgentDetails)
	if item == nil {
		return nil
	}

	sub, err := convert(item)
	if err != nil {
		return nil
	}

	return sub
}

// GetAgentDetailStrings get all the values for the x-agent-details sub resource as string
func GetAgentDetailStrings(h handler) map[string]string {
	details := GetAgentDetails(h)
	if details == nil {
		return nil
	}

	strMap := make(map[string]string)

	for k, v := range details {
		strMap[k] = fmt.Sprint(v)
	}
	return strMap
}

// GetAgentDetailsValue gets a single string value fom the x-agent-details sub resource.
// Returns nil for error if x-agent-details does not exist.
// Returns errors if unable to perform type conversion.
// Returns an empty string if the value does not exist, or if there is an error.
func GetAgentDetailsValue(h handler, key string) (string, error) {
	item := h.GetSubResource(defs.XAgentDetails)
	if item == nil {
		return "", nil
	}

	sub, err := convert(item)
	if err != nil {
		return "", err
	}

	item, ok := sub[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in %s", key, defs.XAgentDetails)
	}

	switch v := item.(type) {
	case int:
		return fmt.Sprintf("%d", v), nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf(
			"%s keys should be a string or int. Received type %T for key %s",
			defs.XAgentDetails,
			v,
			key,
		)
	}
}

// SetAgentDetailsKey sets a key value pair in the x-agent-details sub resource. If x-agent-details does not exist, it is created.
// If value is not a string or an int, an error will be returned.
func SetAgentDetailsKey(h handler, key, value string) error {
	item := h.GetSubResource(defs.XAgentDetails)
	if item == nil {
		h.SetSubResource(defs.XAgentDetails, map[string]interface{}{key: value})
		return nil
	}

	sub, err := convert(item)
	if err != nil {
		return err
	}

	sub[key] = value

	h.SetSubResource(defs.XAgentDetails, sub)
	return nil
}

// SetAgentDetails creates a new x-agent-details sub resource for the given resource.
func SetAgentDetails(h handler, details map[string]interface{}) {
	h.SetSubResource(defs.XAgentDetails, details)
}

func convert(item interface{}) (map[string]interface{}, error) {
	switch v := item.(type) {
	case map[string]interface{}:
		return v, nil
	default:
		return nil, fmt.Errorf(
			"unable to convert %s to type 'AgentDetails'. Received type %T",
			defs.XAgentDetails,
			item,
		)
	}
}
