package v1

import (
	"encoding/json"
	"strings"
)

const (
	ResourceDeleting = "DELETING"
	Inactive         = "inactive"
	Active           = "active"
)

// Meta interface for API Server resource metadata
type Meta interface {
	GetName() string
	GetGroupVersionKind() GroupVersionKind
	GetMetadata() Metadata
	SetScopeName(string)
	GetAttributes() map[string]string
	SetAttributes(map[string]string)
	GetTags() []string
	SetTags([]string)
	GetSubResource(key string) interface{}
	SetSubResource(key string, resource interface{})
	GetReferenceByGVK(GroupVersionKind) Reference
}

// ResourceMeta metadata for a ResourceInstance
type ResourceMeta struct {
	GroupVersionKind
	Name  string `json:"name,omitempty"`
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

// SetScopeName sets the name of a resource
func (rm *ResourceMeta) SetScopeName(name string) {
	rm.Metadata.Scope.Name = name
}

// GetMetadata gets the resource metadata
func (rm *ResourceMeta) GetMetadata() Metadata {
	if rm == nil {
		return Metadata{}
	}

	return rm.Metadata
}

// GetSelfLink gets the resource metadata selflink
func (rm *ResourceMeta) GetSelfLink() string {
	if rm == nil {
		return ""
	}

	// return the self lnk if we have it
	if rm.GetMetadata().SelfLink != "" {
		return rm.Metadata.SelfLink
	}

	if kindLink := rm.GetKindLink(); kindLink != "" {
		return strings.Join([]string{kindLink, rm.Name}, "/")
	}
	return ""
}

// GetKindLink gets the link to resource kind
func (rm *ResourceMeta) GetKindLink() string {
	if rm == nil {
		return ""
	}

	// can't continue if group kind or version are blank
	if rm.Group == "" || rm.Kind == "" || rm.APIVersion == "" {
		return ""
	}

	// empty string to prepend with /
	pathItems := []string{"", rm.Group, rm.APIVersion}

	plural, _ := GetPluralFromKind(rm.Kind)

	scope, ok := GetScope(rm.GetGroupVersionKind().GroupKind)
	if ok && scope != "" {
		scopePlural, _ := GetPluralFromKind(scope)
		pathItems = append(pathItems, []string{scopePlural, rm.Metadata.Scope.Name}...)
	}

	pathItems = append(pathItems, plural)

	return strings.Join(pathItems, "/")
}

// GetGroupVersionKind gets the group, version, and kind of the resource
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
	if rm == nil || rm.SubResources == nil {
		return nil
	}
	return rm.SubResources[key]
}

// SetSubResource saves a value to a sub resource by name and overrides the current value.
// To update a SubResource first call GetSubResource and modify it, then save it.
func (rm *ResourceMeta) SetSubResource(name string, value interface{}) {
	if rm == nil {
		return
	}

	if rm.SubResources == nil {
		rm.SubResources = make(map[string]interface{})
	}
	rm.SubResources[name] = value
}

// GetReferenceByGVK returns the first found reference that matches the GroupKind argument.
func (rm *ResourceMeta) GetReferenceByGVK(gvk GroupVersionKind) Reference {
	for _, ref := range rm.Metadata.References {
		if ref.Group == gvk.Group && ref.Kind == gvk.Kind {
			return ref
		}
	}
	return Reference{}
}

// MarshalJSON marshals the ResourceMeta to properly set the SubResources
func (rm *ResourceMeta) MarshalJSON() ([]byte, error) {
	rawSubs := map[string]interface{}{}
	subResources := rm.SubResources

	if subResources != nil {
		for key, value := range subResources {
			rawSubs[key] = value
		}
	}

	// create an alias for *ResourceMeta to avoid a circular reference while marshalling.
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

// UnmarshalJSON unmarshalls the ResourceMeta to properly set the SubResources
func (rm *ResourceMeta) UnmarshalJSON(data []byte) error {
	type Alias ResourceMeta
	// create an alias for *ResourceMeta to avoid a circular reference while unmarshalling.
	v := &struct{ *Alias }{
		Alias: (*Alias)(rm),
	}

	// unmarshal data to the alias. The SubResources will not be unmarshalled since they are not defined.
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	bts, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// all contains all the defined keys of ResourceMeta. The keys will be used to identify the values
	// that do not belong in the SubResources map.
	all := map[string]interface{}{}
	err = json.Unmarshal(bts, &all)
	if err != nil {
		return err
	}

	// unmarshal data again to a map[string]interface{} to get all the values and the unique sub resources
	rawSubs := map[string]interface{}{}
	err = json.Unmarshal(data, &rawSubs)
	if err != nil {
		return err
	}

	// all contains all keys but the sub resources. rawSubs contains all keys, but should only contain the subresource keys.
	// delete the keys from subs that are not sub resource keys
	for k := range all {
		delete(rawSubs, k)
	}
	delete(rawSubs, "owner")
	delete(rawSubs, "spec")

	if len(rawSubs) > 0 && rm.SubResources == nil {
		rm.SubResources = make(map[string]interface{})
	}

	for k, v := range rawSubs {
		rm.SubResources[k] = v
	}

	return nil
}
