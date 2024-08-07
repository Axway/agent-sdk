package errors

// Generic Agent Errors
var (
	ErrInitServicesNotReady      = New(1001, "failed to initialize. Services are not ready")
	ErrStartingAgentStatusUpdate = Newf(1004, "error starting %s update")
	ErrStartingVersionChecker    = Newf(1005, "%s. No version to compare for upgrade")
	ErrGrpcConnection            = New(1007, "grpc client is not connected to central")
	ErrHarvesterConnection       = New(1008, "harvester client is not connected to central")
)
