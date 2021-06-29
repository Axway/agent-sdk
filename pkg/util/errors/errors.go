package errors

// Generic Agent Errors
var (
	ErrInitServicesNotReady      = New(1001, "failed to initialize. Services are not ready")
	ErrTimeoutServicesNotReady   = New(1002, "failed with timeout error.  Services are not ready")
	ErrPeriodicCheck             = Newf(1003, "%s failed.  Services are not ready")
	ErrStartingAgentStatusUpdate = New(1004, "error starting %s update")
)
