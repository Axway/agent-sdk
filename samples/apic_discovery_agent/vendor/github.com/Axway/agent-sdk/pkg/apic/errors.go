package apic

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Errors hit when validating AMPLIFY Central connectivity
var (
	ErrCentralConfig      = errors.New(1100, "configuration error for AMPLIFY Central")
	ErrEnvironmentQuery   = errors.New(1101, "error sending request to AMPLIFY Central. Check configuration for CENTRAL_ENVIRONMENT")
	ErrTeamNotFound       = errors.Newf(1102, "could not find team (%s) in AMPLIFY Central. Check configuration for CENTRAL_TEAM")
	ErrNetwork            = errors.New(1110, "error connecting to AMPLIFY Central. Check docs.axway.com for more info on this error code")
	ErrRequestQuery       = errors.New(1120, "error making a request to AMPLIFY")
	ErrAuthenticationCall = errors.New(1130, "error getting authentication token. Check AMPLIFY Central auth configuration (CENTRAL_AUTH_*) and network configuration for agent on docs.axway.com")
	ErrAuthentication     = errors.New(1131, "authentication token was not valid. Check AMPLIFY Central auth configuration (CENTRAL_AUTH_*)")
)

// Errors hit when calling different AMPLIFY APIs
var (
	ErrNoAddressFound = errors.Newf(1140, "could not find the subscriber (%s) email address")
	// Subscription APIs
	ErrSubscriptionQuery        = errors.New(1141, "error connecting to AMPLIFY Central for subscriptions")
	ErrSubscriptionResp         = errors.Newf(1142, "unexpected response code (%d) from AMPLIFY Central for subscription")
	ErrSubscriptionSchemaCreate = errors.New(1143, "error creating/updating subscription schema in AMPLIFY Central")
	ErrSubscriptionSchemaResp   = errors.Newf(1144, "unexpected response code (%d) when creating a subscription schema in AMPLIFY Central")

	// APIs related to webhooks
	ErrCreateWebhook = errors.New(1145, "unable to create webhook")
	ErrCreateSecret  = errors.New(1146, "unable to create secret")

	ErrGetSubscriptionDefProperties       = errors.New(1155, "error getting subscription definition properties in AMPLIFY Central")
	ErrUpdateSubscriptionDefProperties    = errors.New(1156, "error updating subscription definition properties in AMPLIFY Central")
	ErrGetCatalogItemServerInfoProperties = errors.New(1157, "error getting catalog item API server info properties")
	ErrSubscriptionManagerDown            = errors.New(1158, "subscription manager is not running")

	// Service body builer
	ErrSetSpecEndPoints = errors.New(1160, "error getting endpoints for the API specification")	
)
