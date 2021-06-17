package agent

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Errors hit when validating AMPLIFY Central connectivity
var (
	ErrUnsupportedAgentType      = errors.New(1000, "unsupported agent type")
	ErrDeletingService           = errors.Newf(1161, "error deleting API Service for catalog item %s in AMPLIFY Central")
	ErrDeletingCatalogItem       = errors.Newf(1162, "error deleting catalog item %s in AMPLIFY Central")
	ErrUnableToGetAPIV1Resources = errors.Newf(1163, "error retrieving API Service resource instances for %s")
)
