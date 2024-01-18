package sampling

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Config errors
var (
	ErrGlobalSamplingCfg = errors.New(1520, "the global sampling config has not been initialized")
	ErrSamplingCfg       = errors.Newf(1521, "sampling percentage must be between 0 and %v.  Setting sampling percentage to default value of %v percent")
)
