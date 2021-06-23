package v1

import "encoding/json"

var metaKeys map[string]struct{} = map[string]struct{}{
	"group":      {},
	"apiVersion": {},
	"kind":       {},
	"name":       {},
	"title":      {},
	"metadata":   {},
	"attributes": {},
	"tags":       {},
}

// ResourceInstance API Server generic resource structure.
type ResourceInstance struct {
	ResourceMeta

	Owner interface{} `json:"owner,omitempty"`
	// Resource instance specs.
	Spec map[string]interface{} `json:"spec"`

	RawResource json.RawMessage
}

//UnmarshalJSON - custom unmarshaler for ResourceInstance struct to additionally use a custom subscriptionField
func (ri *ResourceInstance) UnmarshalJSON(data []byte) error {
	type Alias ResourceInstance // Create an intermediate type to unmarshal the base attributes

	if err := json.Unmarshal(data, &struct{ *Alias }{Alias: (*Alias)(ri)}); err != nil {
		return err
	}

	ri.RawResource = data
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
