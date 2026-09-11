package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type marketplaceHandler struct{}

func (m *marketplaceHandler) shouldProcessPending(status *v1.ResourceStatus, state string) bool {
	return status.Level == prov.Pending.String() && state != v1.ResourceDeleting
}

// shouldProvisionWebhook returns true when the resource is pending and a provisioning webhook is
// configured for this resource type - i.e. this event should be dispatched to the webhook instead of
// the agent's own registered Provisioning implementation.
func (m *marketplaceHandler) shouldProvisionWebhook(status *v1.ResourceStatus, state string, cfg config.ProvisioningWebhookEndpointConfig) bool {
	return m.shouldProcessPending(status, state) && cfg.IsConfigured()
}

func (m *marketplaceHandler) shouldIgnore(action proto.Event_Type, meta *proto.EventMeta) bool {
	if meta == nil {
		return false
	}
	return action == proto.Event_CREATED ||
		(action == proto.Event_SUBRESOURCEUPDATED && meta.Subresource != "status")
}

// shouldProcessDeleting returns true when the resource is in a deleting state and has finalizers
func (m *marketplaceHandler) shouldProcessDeleting(status *v1.ResourceStatus, state string, finalizers []v1.Finalizer) bool {
	return status.Level == prov.Success.String() && state == v1.ResourceDeleting && len(finalizers) > 0
}

// shouldDeprovisionWebhook returns true when the resource is in a deleting state (with finalizers) and a
// provisioning webhook is configured for this resource type - i.e. this event should be dispatched to the
// webhook instead of the agent's own registered Provisioning implementation.
func (m *marketplaceHandler) shouldDeprovisionWebhook(status *v1.ResourceStatus, state string, finalizers []v1.Finalizer, cfg config.ProvisioningWebhookEndpointConfig) bool {
	return m.shouldProcessDeleting(status, state, finalizers) && cfg.IsConfigured()
}

func (m *marketplaceHandler) shouldProcessForAgent(status *v1.ResourceStatus, state string) bool {
	return status.Level == prov.Success.String() && state != v1.ResourceDeleting
}
