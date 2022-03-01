package provisioning

// RequestStatus - holds info about the status of the request
type RequestStatus interface{}

type requestStatus struct {
	RequestStatus
	status     status
	message    string
	properties map[string]string
}

// RequestStatusBuilder - builder to create new request status
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

// NewRequestStatusBuilder - create a request status builder
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

// SetMessage - set the request status message
func (r *requestStatusBuilder) SetMessage(message string) RequestStatusBuilder {
	r.status.message = message
	return r
}

// Success - set the request status as a success
func (r *requestStatusBuilder) Success() RequestStatus {
	r.status.status = Success
	return r.status
}

// Failed - set the request status as failed
func (r *requestStatusBuilder) Failed() RequestStatus {
	r.status.status = Failed
	return r.status
}
