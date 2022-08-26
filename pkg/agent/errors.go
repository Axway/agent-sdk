package agent

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Errors hit when validating Amplify Central connectivity
var (
	ErrDeletingService             = errors.Newf(1161, "error deleting API Service %s in Amplify Central")
	ErrDeletingServiceInstanceItem = errors.Newf(1162, "error deleting API Service Instance %s in Amplify Central")
	ErrUnableToGetAPIV1Resources   = errors.Newf(1163, "error retrieving API Service resource instances for %s")
)
