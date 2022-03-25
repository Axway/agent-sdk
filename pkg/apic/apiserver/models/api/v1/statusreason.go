package v1

// ResourceStatus struct to represent status
type ResourceStatus struct {
	Level string `json:"level"`
	// Details of the warning.
	Reasons []ResourceStatusReason `json:"reasons,omitempty"`
}

// ResourceStatusReason struct for reason oneOfs
type ResourceStatusReason struct {
	Type string `json:"type"`
	// Details of the warning.
	Detail string `json:"detail"`
	// Time when the update occurred.
	Timestamp Time                   `json:"timestamp"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}
