package v1

// Meta
type Meta interface {
	GetName() string
	GetGroupVersionKind() GroupVersionKind
	GetMetadata() Metadata
	GetAttributes() map[string]string
	SetAttributes(map[string]string)
	GetTags() []string
	SetTags([]string)
}

// ResourceMeta metadata for a ResourceInstance
type ResourceMeta struct {
	GroupVersionKind
	Name     string   `json:"name"`
	Title    string   `json:"title,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
	// Custom attributes added to objects.
	Attributes map[string]string `json:"attributes,omitempty"`
	// List of tags.
	Tags []string `json:"tags,omitempty"`
	// Finalizer on the API server resource
	Finalizers []Finalizer
}

// GetName -
func (rm *ResourceMeta) GetName() string {
	if rm == nil {
		return ""
	}

	return rm.Name
}

// SetName -
func (rm *ResourceMeta) SetName(name string) {
	rm.Name = name
}

// GetMetadata -
func (rm *ResourceMeta) GetMetadata() Metadata {
	if rm == nil {
		return Metadata{}
	}

	return rm.Metadata
}

// GetGroupVersionKind -
func (rm *ResourceMeta) GetGroupVersionKind() GroupVersionKind {
	if rm == nil {
		return GroupVersionKind{}
	}

	return rm.GroupVersionKind
}

// GetAttributes -
func (rm *ResourceMeta) GetAttributes() map[string]string {
	if rm == nil {
		return map[string]string{}
	}

	return rm.Attributes
}

// SetAttributes -
func (rm *ResourceMeta) SetAttributes(attrs map[string]string) {
	if rm == nil {
		return
	}

	rm.Attributes = attrs
}

// GetTags -
func (rm *ResourceMeta) GetTags() []string {
	if rm == nil {
		return []string{}
	}

	return rm.Tags
}

// SetTags -
func (rm *ResourceMeta) SetTags(tags []string) {
	if rm == nil {
		return
	}

	rm.Tags = tags
}
