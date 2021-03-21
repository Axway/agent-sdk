package errors

// Generic Agent Errors
var (
	ErrInitServicesNotReady    = New(1001, "failed to initialize. Services are not ready")
	ErrTimeoutServicesNotReady = New(1002, "failed with timeout error.  Services are not ready")
	ErrHealthCheck             = New(1003, "Periodic health checker failed.  Services are not ready")
)
