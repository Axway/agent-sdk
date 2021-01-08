package config

import configerrors "github.com/Axway/agent-sdk/pkg/util/errors"

// Errors hit when validating or parsing config
var (
	ErrBadConfig                 = configerrors.Newf(1401, "error with config %v, please set and/or check its value")
	ErrEnvConfigOverride         = configerrors.New(1402, "error in overriding configuration using environment variables")
	ErrStatusHealthCheckPeriod   = configerrors.New(1403, "invalid value for statusHealthCheckPeriod. Value must be between 1 and 5 minutes")
	ErrStatusHealthCheckInterval = configerrors.New(1404, "invalid value for statusHealthCheckInterval. Value must be between 30 seconds and 5 minutes")
	ErrReadingKeyFile            = configerrors.Newf(1405, "could not read the %v key file %v")
)
