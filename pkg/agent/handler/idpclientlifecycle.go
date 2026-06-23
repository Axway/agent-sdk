package handler

import (
	"context"
	"encoding/json"
	"fmt"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	oktaClientIDsAgentDetail = "oktaClientIDs"
	tokenURLAgentDetail      = "tokenURL"
)

// persistIDPClientOnManagedApplication stores IDP client IDs on the ManagedApplication x-agent-details
// so they can be cleaned up if the app is deleted before credential-level cleanup runs.
func persistIDPClientOnManagedApplication(logger log.FieldLogger, client client, app *management.ManagedApplication, clientID, tokenURL string) error {
	if app == nil || clientID == "" {
		return nil
	}

	details := util.GetAgentDetails(app)
	if details == nil {
		details = make(map[string]interface{})
	}

	existing := extractClientIDs(details[oktaClientIDsAgentDetail])
	for _, id := range existing {
		if id == clientID {
			return nil
		}
	}
	existing = append(existing, clientID)

	details[oktaClientIDsAgentDetail] = existing
	if tokenURL != "" {
		details[tokenURLAgentDetail] = tokenURL
	}

	if err := client.CreateSubResource(app.ResourceMeta, map[string]interface{}{defs.XAgentDetails: details}); err != nil {
		return fmt.Errorf("could not persist IDP client reference on managed application %s: %w", app.Name, err)
	}

	if logger != nil {
		logger.WithField("clientID", clientID).Trace("persisted IDP client reference on managed application")
	}

	return nil
}

// cleanupManagedApplicationIDPClients removes any tracked IDP clients for a ManagedApplication.
func cleanupManagedApplicationIDPClients(ctx context.Context, logger log.FieldLogger, registry oauth.IdPRegistry, app *management.ManagedApplication) error {
	if registry == nil || app == nil {
		return nil
	}

	details := util.GetAgentDetails(app)
	if details == nil {
		return nil
	}

	clientIDs := extractClientIDs(details[oktaClientIDsAgentDetail])
	if len(clientIDs) == 0 {
		return nil
	}

	tokenURL := util.ToString(details[tokenURLAgentDetail])
	if tokenURL == "" {
		logger.Warn("no tokenURL in managed application x-agent-details, skipping IDP app cleanup")
		return nil
	}

	provider, err := registry.GetProviderByTokenEndpoint(ctx, tokenURL)
	if err != nil || provider == nil {
		logger.WithField("tokenURL", tokenURL).Warn("no IDP provider registered for tokenURL, skipping IDP app cleanup")
		return nil
	}

	for _, clientID := range clientIDs {
		if err := provider.UnregisterClient(clientID, "", "", nil, ""); err != nil {
			return fmt.Errorf("cleanupManagedApplicationIDPClients: failed for client %s: %w", clientID, err)
		}
		logger.WithField("clientID", clientID).Info("IDP client unregistered")
	}

	return nil
}

func extractClientIDs(raw interface{}) []string {
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []interface{}:
		ids := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				ids = append(ids, s)
			}
		}
		return ids
	case string:
		if v == "" {
			return nil
		}
		var ids []string
		if err := json.Unmarshal([]byte(v), &ids); err == nil {
			return ids
		}
	}

	return nil
}
