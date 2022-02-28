package provisioning

import "fmt"

const statusSetError = "can not set status as it's already been set"

// RequestStatus - holds info about the status of the request
type RequestStatus struct {
	status  status
	message string
}

// RequestStatusBuilder - builder to create new request status
type RequestStatusBuilder interface {
	Success() RequestStatusBuilder
	Failed(message string) RequestStatusBuilder
	Process() (*RequestStatus, error)
}

type requestStatusBuilder struct {
	err    error
	status *RequestStatus
}

// NewRequestStatusBuilder - create a request status builder
func NewRequestStatusBuilder() RequestStatusBuilder {
	return &requestStatusBuilder{
		status: &RequestStatus{},
	}
}

// Process - process the builder, returning errors
func (r *requestStatusBuilder) Process() (*RequestStatus, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.status, r.err
}

// Success - set the request status as a success
func (r *requestStatusBuilder) Success() RequestStatusBuilder {
	if r.err != nil {
		return r
	}

	if r.status.status != 0 {
		r.err = fmt.Errorf(statusSetError)
		return r
	}

	r.status.status = Success
	return r
}

// Failed - set the request status as failed and include a message
func (r *requestStatusBuilder) Failed(message string) RequestStatusBuilder {
	if r.err != nil {
		return r
	}

	if r.status.status != 0 {
		r.err = fmt.Errorf(statusSetError)
		return r
	}

	r.status.status = Failed
	r.status.message = message
	return r
}
