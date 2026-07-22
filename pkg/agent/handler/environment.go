package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type environmentHandler struct {
	agentCacheManager agentcache.Manager
	credentialConfig  config.CredentialConfig
	envName           string
}

// NewEnvironmentHandler creates a Handler for Environments.
func NewEnvironmentHandler(agentCacheManager agentcache.Manager, credentialConfig config.CredentialConfig, envName string) Handler {
	return &environmentHandler{
		agentCacheManager: agentCacheManager,
		credentialConfig:  credentialConfig,
		envName:           envName,
	}
}

func (c *environmentHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	if event.Payload.Metadata.Scope.Name != c.envName {
		return false
	}
	// verify that action is subresource updated and meta subsresource is environment policy
	action := GetActionFromContext(ctx)
	if action != proto.Event_SUBRESOURCEUPDATED || event.Metadata.Subresource != management.EnvironmentPoliciesSubResourceName {
		return false
	}

	return true
}

// GetAPIServerFields returns the fields needed to process the given event. This handler only ever
// reacts to a subresource update of the environment's policies, so that's all it needs.
func (c *environmentHandler) GetAPIServerFields(ctx context.Context, event *proto.Event) []string {
	action := GetActionFromContext(ctx)
	if action != proto.Event_SUBRESOURCEUPDATED || event.Metadata.Subresource != management.EnvironmentPoliciesSubResourceName {
		return nil
	}
	return []string{"name", "metadata.id", event.Metadata.Subresource}
}

func (c *environmentHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	log := getLoggerFromContext(ctx).WithComponent("environmentHandler")
	env := &management.Environment{}
	err := env.FromInstance(resource)
	if err != nil {
		log.WithError(err).Error("could not handle environment resource")
		return nil
	}

	// Set up credential config from environment resource policies
	c.credentialConfig.SetShouldDeprovisionExpired(env.Policies.Credentials.Expiry.Action == "deprovision")
	c.credentialConfig.SetExpirationDays(int(env.Policies.Credentials.Expiry.Period))

	return nil
}
