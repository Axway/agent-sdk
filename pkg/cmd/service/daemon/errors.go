package daemon

import "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"

// Errors that are hit while managing an Agent's service
var (
	ErrUnsupportedSystem = errors.New(1900, "unsupported system")
	ErrNeedSystemd       = errors.New(1901, "need systemd to install the service")
	ErrRootPrivileges    = errors.New(1902, "requires root user privileges. Possibly use 'sudo'")
	ErrAlreadyInstalled  = errors.Newf(1903, "%s service has already been installed")
	ErrCurrentlyRunning  = errors.Newf(1904, "%s service is running and cannot be removed")
	ErrNotInstalled      = errors.Newf(1905, "%s service is not installed")
	ErrAlreadyRunning    = errors.Newf(1906, "%s service is already running")
	ErrAlreadyStopped    = errors.Newf(1907, "%s service is already stopped")
)
