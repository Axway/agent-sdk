package errors

// Generic Agent Errors
var (
	ErrServicesNotReady = New(1001, "failed with timeout error.  Services are not ready")
)
