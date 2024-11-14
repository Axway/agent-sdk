package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Interface describes API Server & catalog resources
type Interface interface {
	Meta
	AsInstance() (*ResourceInstance, error)
	FromInstance(from *ResourceInstance) error
}

// ResourceInstance API Server generic resource structure.
type ResourceInstance struct {
	ResourceMeta
	Owner *Owner `json:"owner"`
	// Resource instance specs.
	Spec        map[string]interface{} `json:"spec"`
	rawResource json.RawMessage
}

// UnmarshalJSON - custom unmarshaler for ResourceInstance struct to additionally use a custom subscriptionField
func (ri *ResourceInstance) UnmarshalJSON(data []byte) error {
	type Alias ResourceInstance // Create an intermediate type to unmarshal the base attributes
	if err := json.Unmarshal(data, &struct{ *Alias }{Alias: (*Alias)(ri)}); err != nil {
		return err
	}

	// unmarshall the rest of the resources here, and set them on the ResourceInstance manually
	out := map[string]interface{}{}
	err := json.Unmarshal(data, &out)
	if err != nil {
		return err
	}

	if out["spec"] != nil {
		v, ok := out["spec"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("spec is not a map[string]interface{}")
		}
		ri.Spec = v
	}

	if out["owner"] != nil {
		var err error
		ri.Owner = &Owner{}
		bts, err := json.Marshal(out["owner"])
		if err != nil {
			return err
		}
		err = json.Unmarshal(bts, ri.Owner)
		if err != nil {
			return err
		}
	}

	// clean up any unnecessary chars from json byte array
	byteBuf := bytes.Buffer{}
	err = json.Compact(&byteBuf, data)
	if err != nil {
		return err
	}

	ri.rawResource = byteBuf.Bytes()
	return nil
}

// MarshalJSON - custom marshaller for ResourceInstance to save the rawResource json to unmarshal specific types
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

	rawInstance["spec"] = ri.Spec
	rawInstance["owner"] = ri.Owner

	keysToDelete := make([]string, 0)
	for key := range rawStruct {
		_, ok := rawInstance[key]
		if !ok {
			keysToDelete = append(keysToDelete, key)
		}
	}

	// override the rawStruct map with the values from the rawInstance map
	for key, value := range rawInstance {
		rawStruct[key] = value
	}

	// remove deleted sub-resources
	for _, key := range keysToDelete {
		delete(rawStruct, key)
	}

	// return the marshal of the rawStruct
	return json.Marshal(rawStruct)
}

// AsInstance returns the ResourceInstance
func (ri *ResourceInstance) AsInstance() (*ResourceInstance, error) {
	return ri, nil
}

// FromInstance sets the underlying ResourceInstance
func (ri *ResourceInstance) FromInstance(from *ResourceInstance) error {
	*ri = *from

	return nil
}

// GetRawResource gets the resource as bytes
func (ri *ResourceInstance) GetRawResource() json.RawMessage {
	return ri.rawResource
}
