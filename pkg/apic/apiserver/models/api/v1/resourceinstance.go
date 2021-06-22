package v1

import "encoding/json"

// ResourceInstance API Server generic resource structure.
type ResourceInstance struct {
	ResourceMeta

	Owner *struct{} `json:"owner,omitempty"`
	// Resource instance specs.
	Spec map[string]interface{} `json:"spec"`

	SubResources map[string]json.RawMessage
}

//UnmarshalJSON - custom unmarshaler for ResourceInstance struct to additionally use a custom subscriptionField
func (ri *ResourceInstance) UnmarshalJSON(data []byte) error {
	type Alias ResourceInstance // Create an intermediate type to unmarshal the base attributes

	if err := json.Unmarshal(data, &struct{ *Alias }{Alias: (*Alias)(ri)}); err != nil {
		return err
	}

	var allFields interface{}
	json.Unmarshal(data, &allFields)
	b := allFields.(map[string][]byte)

	for key, value := range b {
		if key != "owner" && key != "spec" {
			ri.SubResources[key] = value
		}
	}
	return nil
}

// AsInstance -
func (ri *ResourceInstance) AsInstance() (*ResourceInstance, error) {
	return ri, nil
}

// FromInstance -
func (ri *ResourceInstance) FromInstance(from *ResourceInstance) error {
	*ri = *from

	return nil
}

//Interface -
type Interface interface {
	Meta
	AsInstance() (*ResourceInstance, error)
	FromInstance(from *ResourceInstance) error
}
