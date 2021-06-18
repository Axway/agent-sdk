package v1

// ResourceInstance API Server generic resource structure.
type ResourceInstance struct {
	ResourceMeta
	// GENERATE: The following code has been modified after code generation
	//  	Owner struct{} `json:"owner"`
	Owner *struct{} `json:"owner,omitempty"`
	// Resource instance specs.
	Spec map[string]interface{} `json:"spec"`
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
