package provisioning

import apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

// RequestStatus - holds info about the Status of the request
type RequestStatus interface {
	GetStatus() Status
	GetMessage() string
}

type requestStatus struct {
	RequestStatus
	status     Status // -> Level
	message    string
	properties map[string]string
}

func (rs *requestStatus) GetStatus() Status {
	return rs.status
}

func (rs *requestStatus) GetMessage() string {
	return rs.message
}

// RequestStatusBuilder - builder to create new request Status
type RequestStatusBuilder interface {
	Success() RequestStatus
	Failed() RequestStatus
	SetMessage(message string) RequestStatusBuilder
	SetProperties(map[string]string) RequestStatusBuilder
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
func (r *requestStatusBuilder) SetProperties(properties map[string]string) RequestStatusBuilder {
	r.status.properties = properties
	return r
}

// Failed - add a property to be sent back to the resource
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
	r.status.status = Failed
	return r.status
}

// NewStatusReason converts a RequestStatus into a ResourceStatus
func NewStatusReason(r RequestStatus) apiv1.ResourceStatus {
	msg := r.GetMessage()
	var reasons []apiv1.ResourceStatusReason
	if msg != "" {
		reasons = make([]apiv1.ResourceStatusReason, 0)
		reasons[0] = apiv1.ResourceStatusReason{
			Type:      r.GetStatus().String(),
			Detail:    msg,
			Timestamp: apiv1.Time{},
		}
	}

	status := apiv1.ResourceStatus{
		Level:   msg,
		Reasons: reasons,
	}

	return status
}
