package log

import configerrors "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"

// Log Config Errors
var (
	ErrInvalidLogConfig = configerrors.Newf(1410, "logging configuration error - %v does not meet criteria (%v)")
)
