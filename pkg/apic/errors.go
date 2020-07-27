package apic

import "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/exception"

// Errors hit when validating AMPLIFY Central connectivity
var (
	ErrCentralConfig      = exception.New(1000, "configuration error for AMPLIFY Central")
	ErrEnvironmentQuery   = exception.New(1001, "error sending request to AMPLIFY Central. Check configuration for CENTRAL_ENVIRONMENT")
	ErrNetwork            = exception.New(1010, "error connecting to AMPLIFY Central. Check docs.axway.com for more info on this error code")
	ErrRequestQuery       = exception.Newf(1020, "error making a request to AMPLIFY. %s")
	ErrAuthenticationCall = exception.New(1030, "error getting authentication token. Check AMPLIFY Central auth configuration (CENTRAL_AUTH_*) and network configuration for agent on docs.axway.com")
	ErrAuthentication     = exception.New(1031, "authentication token was not valid. Check AMPLIFY Central auth configuration (CENTRAL_AUTH_*)")
)

// Errors hit when querying different AMPLIFY APIs
var (
	ErrNoAddressFound = exception.Newf(1100, "could not find the subscriber (%s) email address")
)
