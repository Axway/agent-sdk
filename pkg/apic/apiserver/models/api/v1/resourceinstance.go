package v1

// ResourceInstance API Server generic resource structure.
type ResourceInstance struct {
	ResourceMeta
	// Resource instance specs.
	Spec map[string]interface{} `json:"spec"`
}

func (ri *ResourceInstance) AsInstance() (*ResourceInstance, error) {
	return ri, nil
}

func (ri *ResourceInstance) FromInstance(from *ResourceInstance) error {
	*ri = *from

	return nil
}

type Interface interface {
	Meta
	AsInstance() (*ResourceInstance, error)
	FromInstance(from *ResourceInstance) error
}
