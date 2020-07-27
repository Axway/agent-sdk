package apic

import "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/exception"

// Errors hit when validating AMPLIFY Central connectivity
var (
	ErrCentralConfig      = exception.New(1000, "configuration error for AMPLIFY Central")
	ErrNetwork            = exception.New(1010, "error connecting to AMPLIFY Central. Check docs.axway.com for more info on this error code")
	ErrAuthenticationCall = exception.New(1020, "error getting authentication token. Check AMPLIFY Central auth configuration (CENTRAL_AUTH_*) and network configuration for agent on docs.axway.com")
	ErrAuthentication     = exception.New(1021, "authentication token was not valid. Check AMPLIFY Central auth configuration (CENTRAL_AUTH_*)")
	ErrEnvironmentQuery   = exception.New(1030, "error sending request to AMPLIFY Central. Check configuration for CENTRAL_ENVIRONMENT")
)
