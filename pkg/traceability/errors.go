package traceability

import "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"

// Config errors
var (
	ErrSettingProxy            = errors.New(1500, "could not set proxy url environment variable")
	ErrFailedPublishing        = errors.New(1501, "failed to publish events")
	ErrClosingCondorConnection = errors.Newf(1502, "error closing connection to traceability host %s, reconnecting...")
	ErrHTTPNotConnected        = errors.New(1503, "http transport is not connected")
	ErrJSONEncodeFailed        = errors.New(1504, "failed to encode the json content")
	ErrInvalidConfig           = errors.Newf(1505, "invalid traceability config. Config error: %s)")
)
