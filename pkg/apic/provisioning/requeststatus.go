package provisioning

import "fmt"

const statusSetError = "can not set status as it's already been set"

type RequestStatus struct {
	status  ProvisioningStatus
	message string
}

type RequestStatusBuilder interface {
	Success() RequestStatusBuilder
	Failed(message string) RequestStatusBuilder
	Process() (*RequestStatus, error)
}

type requestStatusBuilder struct {
	err    error
	status *RequestStatus
}

func NewRequestStatusBuilder() RequestStatusBuilder {
	return &requestStatusBuilder{
		status: &RequestStatus{},
	}
}

func (r *requestStatusBuilder) Process() (*RequestStatus, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.status, r.err
}

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
