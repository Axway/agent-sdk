package provisioning

import (
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// RequestStatus - holds info about the Status of the request
type RequestStatus interface {
	// GetReasons returns the Status reasons
	GetReasons() []v1.ResourceStatusReason
	// GetStatus returns the Status level
	GetStatus() Status
	// GetMessage returns the status message
	GetMessage() string
	// GetProperties returns additional details about a status.
	GetProperties() map[string]string
}

type requestStatus struct {
	RequestStatus
	status     Status
	message    string
	properties map[string]string
	reasons    []v1.ResourceStatusReason
}

// GetStatus returns the Status level
func (rs *requestStatus) GetReasons() []v1.ResourceStatusReason {
	return rs.reasons
}

// GetStatus returns the Status level
func (rs *requestStatus) GetStatus() Status {
	return rs.status
}

// GetMessage returns the status message
func (rs *requestStatus) GetMessage() string {
	return rs.message
}

// GetProperties returns additional details about a status.
func (rs *requestStatus) GetProperties() map[string]string {
	return rs.properties
}

// RequestStatusBuilder - builder to create new request Status
type RequestStatusBuilder interface {
	// Success - set the status as success
	Success() RequestStatus
	// Failed - set the status as failed
	Failed() RequestStatus
	// RequestStatusBuilder - adds any existing status reasons so they are not lost
	SetCurrentStatusReasons([]v1.ResourceStatusReason) RequestStatusBuilder
	// SetMessage - set the request Status message
	SetMessage(message string) RequestStatusBuilder
	// SetProperties - set the properties of the RequestStatus
	SetProperties(map[string]string) RequestStatusBuilder
	// AddProperty - add a new property on the RequestStatus
	AddProperty(key string, value string) RequestStatusBuilder
}

type requestStatusBuilder struct {
	status *requestStatus
}

// NewRequestStatusBuilder - create a request Status builder
func NewRequestStatusBuilder() RequestStatusBuilder {
	return &requestStatusBuilder{
		status: &requestStatus{
			properties: make(map[string]string),
		},
	}
}

// SetProperties - set the properties to be sent back to the resource
func (r *requestStatusBuilder) SetCurrentStatusReasons(reasons []v1.ResourceStatusReason) RequestStatusBuilder {
	r.status.reasons = reasons
	return r
}

// SetProperties - set the properties to be sent back to the resource
func (r *requestStatusBuilder) SetProperties(properties map[string]string) RequestStatusBuilder {
	r.status.properties = properties
	return r
}

// AddProperty - add a property to be sent back to the resource
func (r *requestStatusBuilder) AddProperty(key, value string) RequestStatusBuilder {
	r.status.properties[key] = value
	return r
}

// SetMessage - set the request Status message
func (r *requestStatusBuilder) SetMessage(message string) RequestStatusBuilder {
	r.status.message = message
	return r
}

// Success - set the request Status as a success
func (r *requestStatusBuilder) Success() RequestStatus {
	r.status.status = Success
	return r.status
}

// Failed - set the request Status as failed
func (r *requestStatusBuilder) Failed() RequestStatus {
	r.status.status = Error
	return r.status
}

// NewStatusReason converts a RequestStatus into a ResourceStatus
func NewStatusReason(r RequestStatus) *v1.ResourceStatus {
	if r == nil {
		return nil
	}

	reasons := r.GetReasons()
	if reasons == nil {
		reasons = []v1.ResourceStatusReason{}
	}

	// append the new reason
	reasons = append(reasons, v1.ResourceStatusReason{
		Type:      r.GetStatus().String(),
		Detail:    r.GetMessage(),
		Timestamp: v1.Time(time.Now()),
	})

	return &v1.ResourceStatus{
		Level:   r.GetStatus().String(),
		Reasons: reasons,
	}
}
