package redaction

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Config errors
var (
	ErrGlobalRedactionCfg = errors.New(1510, "the global redaction config has not been initialized")
	ErrInvalidRegex       = errors.Newf(1511, "could not compile the %s regex value (%v): %v")
)
