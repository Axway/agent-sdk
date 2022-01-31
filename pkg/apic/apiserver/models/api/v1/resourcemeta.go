package v1

import "encoding/json"

// Meta interface for API Server resource metadata
type Meta interface {
	GetName() string
	GetGroupVersionKind() GroupVersionKind
	GetMetadata() Metadata
	GetAttributes() map[string]string
	SetAttributes(map[string]string)
	GetTags() []string
	SetTags([]string)
	GetSubResource(key string) interface{}
	SetSubResource(key string, resource interface{})
}

// ResourceMeta metadata for a ResourceInstance
type ResourceMeta struct {
	GroupVersionKind
	Name  string `json:"name"`
	Title string `json:"title,omitempty"`
	// Metadata the metadata for the resource
	Metadata Metadata `json:"metadata,omitempty"`
	// Custom attributes for a resource.
	Attributes map[string]string `json:"attributes"`
	// List of tags.
	Tags []string `json:"tags"`
	// Finalizer on the API server resource
	Finalizers []Finalizer `json:"finalizers"`
	// SubResources contains all of the unique sub resources that may be added to a resource
	SubResources map[string]interface{} `json:"-"`
}

// GetName gets the name of a resource
func (rm *ResourceMeta) GetName() string {
	if rm == nil {
		return ""
	}

	return rm.Name
}

// SetName sets the name of a resource
func (rm *ResourceMeta) SetName(name string) {
	rm.Name = name
}

// GetMetadata gets the resource metadata
func (rm *ResourceMeta) GetMetadata() Metadata {
	if rm == nil {
		return Metadata{}
	}

	return rm.Metadata
}

// GetGroupVersionKind gets thee group, version, and kind of the resource
func (rm *ResourceMeta) GetGroupVersionKind() GroupVersionKind {
	if rm == nil {
		return GroupVersionKind{}
	}

	return rm.GroupVersionKind
}

// GetAttributes gets the attributes of a resource
func (rm *ResourceMeta) GetAttributes() map[string]string {
	if rm == nil {
		return map[string]string{}
	}

	return rm.Attributes
}

// SetAttributes sets the attributes of a resource
func (rm *ResourceMeta) SetAttributes(attrs map[string]string) {
	if rm == nil {
		return
	}

	rm.Attributes = attrs
}

// GetTags gets the tags of the resource
func (rm *ResourceMeta) GetTags() []string {
	if rm == nil {
		return []string{}
	}

	return rm.Tags
}

// SetTags adds tags to the resource
func (rm *ResourceMeta) SetTags(tags []string) {
	if rm == nil {
		return
	}

	rm.Tags = tags
}

// GetSubResource get a sub resource by name
func (rm *ResourceMeta) GetSubResource(key string) interface{} {
	if rm.SubResources == nil {
		return nil
	}
	return rm.SubResources[key]
}

// SetSubResource saves a value to a sub resource by name and overrides the current value.
// To update a SubResource first call GetSubResource and modify it, then save it.
func (rm *ResourceMeta) SetSubResource(name string, value interface{}) {
	if rm.SubResources == nil {
		rm.SubResources = make(map[string]interface{})
	}
	rm.SubResources[name] = value
}

// MarshalJSON marshals the ResourceMeta to properly set the SubResources
func (rm *ResourceMeta) MarshalJSON() ([]byte, error) {
	rawSubs := map[string]interface{}{}
	subResources := rm.SubResources

	if subResources != nil {
		for key, value := range subResources {
			rawSubs[key] = value
		}

		bts, err := json.Marshal(&rawSubs)
		if err != nil {
			return bts, err
		}
	}

	type Alias ResourceMeta
	v := &struct{ *Alias }{
		Alias: (*Alias)(rm),
	}

	metaBts, err := json.Marshal(v)
	if err != nil {
		return metaBts, err
	}

	rawMeta := map[string]interface{}{}
	err = json.Unmarshal(metaBts, &rawMeta)
	if err != nil {
		return metaBts, nil
	}

	for k, v := range rawSubs {
		rawMeta[k] = v
	}

	return json.Marshal(rawMeta)
}

// UnmarshalJSON unmarshals the ResourceMeta to properly set the SubResources
func (rm *ResourceMeta) UnmarshalJSON(data []byte) error {
	type Alias ResourceMeta
	v := &struct{ *Alias }{
		Alias: (*Alias)(rm),
	}

	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	raw := map[string]interface{}{}
	err = json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	delete(raw, "name")
	delete(raw, "title")
	delete(raw, "metadata")
	delete(raw, "attributes")
	delete(raw, "tags")
	delete(raw, "finalizers")

	if rm.SubResources == nil {
		rm.SubResources = make(map[string]interface{})
	}
	for k, v := range raw {
		rm.SubResources[k] = v
	}

	return nil
}
