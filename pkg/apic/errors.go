package apic

import "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/errors"

// Errors hit when validating AMPLIFY Central connectivity
var (
	ErrCentralConfig      = errors.New(1000, "configuration error for AMPLIFY Central")
	ErrEnvironmentQuery   = errors.New(1001, "error sending request to AMPLIFY Central. Check configuration for CENTRAL_ENVIRONMENT")
	ErrNetwork            = errors.New(1010, "error connecting to AMPLIFY Central. Check docs.axway.com for more info on this error code")
	ErrRequestQuery       = errors.Newf(1020, "error making a request to AMPLIFY. %s")
	ErrAuthenticationCall = errors.New(1030, "error getting authentication token. Check AMPLIFY Central auth configuration (CENTRAL_AUTH_*) and network configuration for agent on docs.axway.com")
	ErrAuthentication     = errors.New(1031, "authentication token was not valid. Check AMPLIFY Central auth configuration (CENTRAL_AUTH_*)")
)

// Errors hit when querying different AMPLIFY APIs
var (
	ErrNoAddressFound = errors.Newf(1100, "could not find the subscriber (%s) email address")
	// Subscription APIs
	ErrSubscriptionQuery        = errors.New(1110, "error connecting to AMPLIFY Central for subscriptions")
	ErrSubscriptionResp         = errors.Newf(1111, "unexpected response code (%d) from AMPLIFY Central for subscription")
	ErrSubscriptionSchemaCreate = errors.New(1112, "error creating/updating subscription schema in AMPLIFY Central")
	ErrSubscriptionSchemaResp   = errors.Newf(1113, "unexpected response code (%d) when creating a subscription schema in AMPLIFY Central")
)
