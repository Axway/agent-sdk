package apic

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Errors hit when validating Amplify Central connectivity
var (
	ErrCentralConfig      = errors.New(1100, "configuration error for Amplify Central")
	ErrEnvironmentQuery   = errors.New(1101, "error sending request to Amplify Central. Check configuration for CENTRAL_ENVIRONMENT")
	ErrTeamNotFound       = errors.Newf(1102, "could not find team (%s) in Amplify Central. Check configuration for CENTRAL_TEAM")
	ErrNetwork            = errors.New(1110, "error connecting to Amplify Central. Check docs.axway.com for more info on this error code")
	ErrRequestQuery       = errors.New(1120, "error making a request to Amplify")
	ErrAuthenticationCall = errors.New(1130, "error getting authentication token. Check Amplify Central auth configuration (CENTRAL_AUTH_*) and network configuration for agent on docs.axway.com")
	ErrAuthentication     = errors.New(1131, "authentication token was not valid. Check Amplify Central auth configuration (CENTRAL_AUTH_*)")
)

// Errors hit when calling different Amplify APIs
var (
	ErrNoAddressFound = errors.Newf(1139, "could not find the subscriber (%s) email address")

	// APIs related to webhooks

	ErrCreateSecret                        = errors.New(1145, "unable to create secret")
	ErrAccessRequestSubscriptionSchemaResp = errors.Newf(1146, "unexpected response code (%d) when creating an access request subscription schema in Amplify Central")

	ErrGetSubscriptionDefProperties    = errors.New(1155, "error getting subscription definition properties in Amplify Central")
	ErrUpdateSubscriptionDefProperties = errors.New(1156, "error updating subscription definition properties in Amplify Central")
	ErrSubscriptionManagerDown         = errors.New(1158, "subscription manager is not running")

	// Service body builder
	ErrSetSpecEndPoints = errors.New(1160, "error getting endpoints for the API specification")
)
