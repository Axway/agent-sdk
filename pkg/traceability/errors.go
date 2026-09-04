package traceability

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Config errors
var (
	ErrHTTPNotConnected       = errors.New(1503, "http transport is not connected")
	ErrJSONEncodeFailed       = errors.New(1504, "failed to encode the json content")
	ErrInvalidConfig          = errors.Newf(1505, "invalid traceability config. Config error: %s")
	ErrInvalidRegex           = errors.Newf(1506, "could not compile the %s regex value (%v): %v")
	ErrTCPProtocolRemoved     = errors.New(1507, "protocol 'tcp' is no longer supported; set output.traceability.protocol to https")
	ErrPort5044Removed        = errors.Newf(1508, "host %s uses port 5044 (tcp/lumberjack), which is no longer supported. Use port 443 with protocol https")
	ErrIngestionHostRemoved   = errors.Newf(1509, "host %s uses a legacy ingestion.* address, which is no longer supported. Use the phoenix.* host for your region")
	ErrNoConnectionConfigured = errors.New(1510, "no connection configured")
	ErrNoActiveConnection     = errors.New(1511, "no active connection")
)
