package v1

import (
	"bytes"
	"encoding/json"
)

// ResourceInstance API Server generic resource structure.
type ResourceInstance struct {
	ResourceMeta

	Owner *Owner `json:"owner,omitempty"`
	// Resource instance specs.
	Spec map[string]interface{} `json:"spec"`

	rawResource json.RawMessage
}

//UnmarshalJSON - custom unmarshaler for ResourceInstance struct to additionally use a custom subscriptionField
func (ri *ResourceInstance) UnmarshalJSON(data []byte) error {
	type Alias ResourceInstance // Create an intermediate type to unmarshal the base attributes
	if err := json.Unmarshal(data, &struct{ *Alias }{Alias: (*Alias)(ri)}); err != nil {
		return err
	}

	// clean up any unnecessary chars from json byte array
	byteBuf := bytes.Buffer{}
	json.Compact(&byteBuf, data)

	ri.rawResource = byteBuf.Bytes()
	return nil
}

//MarshalJSON - custom marshaler for ResourceInstance to save the rawResource json to unmarshal specific types
func (ri *ResourceInstance) MarshalJSON() ([]byte, error) {
	// unmarshal the rawResource to map[string]interface{}
	rawStruct := map[string]interface{}{}
	if ri.rawResource != nil {
		if err := json.Unmarshal(ri.rawResource, &rawStruct); err != nil {
			return []byte{}, err
		}
	}

	// marshal the current resource instance then unmarshal it into map[string]interface{}{}
	type Alias ResourceInstance // Create an intermittent type to marshal the base attributes
	riAlias, err := json.Marshal(&struct{ *Alias }{Alias: (*Alias)(ri)})
	if err != nil {
		return nil, err
	}
	rawInstance := map[string]interface{}{}
	if err := json.Unmarshal(riAlias, &rawInstance); err != nil {
		return []byte{}, err
	}

	// override the rawStruct map with the values from the rawInstance map
	for key, value := range rawInstance {
		rawStruct[key] = value
	}

	// return the marshal of the rawStruct
	return json.Marshal(rawStruct)
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

// GetRawResource -
func (ri *ResourceInstance) GetRawResource() json.RawMessage {
	return ri.rawResource
}

//Interface -
type Interface interface {
	Meta
	AsInstance() (*ResourceInstance, error)
	FromInstance(from *ResourceInstance) error
}
