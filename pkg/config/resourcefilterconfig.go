package config

// ResourceEventType - watch filter event types
type ResourceEventType string

// Resource event types
const (
	ResourceEventCreated ResourceEventType = "created"
	ResourceEventUpdated ResourceEventType = "updated"
	ResourceEventDeleted ResourceEventType = "deleted"
)

// ResourceScope -  scope config for watch resource filter
type ResourceScope struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// ResourceFilter - custom watch filter
type ResourceFilter struct {
	Group            string              `json:"group"` // remove group ? and default to management or allow filter for other groups as well?
	Kind             string              `json:"kind"`
	Name             string              `json:"name"`
	EventTypes       []ResourceEventType `json:"eventTypes"`
	Scope            *ResourceScope      `json:"scope"`
	IsCachedResource bool                `json:"isCachedResource"`
	IsUnscoped       bool                `json:"isUnscoped"`
}
