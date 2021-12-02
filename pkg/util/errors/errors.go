package errors

// Generic Agent Errors
var (
	ErrInitServicesNotReady        = New(1001, "failed to initialize. Services are not ready")
	ErrTimeoutServicesNotReady     = New(1002, "failed with timeout error.  Services are not ready")
	ErrPeriodicCheck               = Newf(1003, "%s failed.  Services are not ready")
	ErrStartingAgentStatusUpdate   = Newf(1004, "error starting %s update")
	ErrStartingVersionChecker      = Newf(1005, "%s. No version to compare for upgrade")
	ErrRegisterSubscriptionWebhook = New(1006, "unable to register subscription webhook")
)
