package traceability

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Config errors
var (
	ErrHTTPNotConnected = errors.New(1503, "http transport is not connected")
	ErrJSONEncodeFailed = errors.New(1504, "failed to encode the json content")
	ErrInvalidConfig    = errors.Newf(1505, "invalid traceability config. Config error: %s")
	ErrInvalidRegex     = errors.Newf(1506, "could not compile the %s regex value (%v): %v")
)
