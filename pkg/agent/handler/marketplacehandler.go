package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type marketplaceHandler struct{}

func (m *marketplaceHandler) shouldProcessPending(status *v1.ResourceStatus, state string) bool {
	return status.Level == prov.Pending.String() && state != v1.ResourceDeleting
}

func (m *marketplaceHandler) shouldIgnoreSubResourceUpdate(action proto.Event_Type, meta *proto.EventMeta) bool {
	if meta == nil {
		return false
	}
	return action == proto.Event_SUBRESOURCEUPDATED && meta.Subresource != "status"
}

// shouldProcessDeleting returns true when the resource is in a deleting state and has finalizers
func (m *marketplaceHandler) shouldProcessDeleting(status *v1.ResourceStatus, state string, finalizers []v1.Finalizer) bool {
	return status.Level == prov.Success.String() && state == v1.ResourceDeleting && len(finalizers) > 0
}

func (m *marketplaceHandler) shouldProcessForTrace(status *v1.ResourceStatus, state string) bool {
	return status.Level == prov.Success.String() && state != v1.ResourceDeleting
}
