package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// Handler interface used by the EventListener to process events.
type Handler interface {
	// Handle receives the type of the event (add, update, delete), event metadata and the API Server resource, if it exists.
	Handle(action proto.Event_Type, eventMetadata *proto.EventMeta, resource *v1.ResourceInstance) error
}

// client is an interface that is implemented by the ServiceClient in apic/client.go.
type client interface {
	GetResource(url string) (*v1.ResourceInstance, error)
	CreateResource(url string, bts []byte) (*v1.ResourceInstance, error)
	UpdateResource(url string, bts []byte) (*v1.ResourceInstance, error)
	CreateSubResourceScoped(scopeKindPlural, resKindPlural string, rm v1.ResourceMeta, subs map[string]interface{}) error
}

func isStatusFound(rs *v1.ResourceStatus) bool {
	if rs == nil || rs.Level == "" {
		return false
	}
	return true
}

func isNotStatusSubResourceUpdate(action proto.Event_Type, meta *proto.EventMeta) bool {
	if meta != nil {
		return (action == proto.Event_SUBRESOURCEUPDATED && meta.Subresource != "status")
	}
	return false
}
