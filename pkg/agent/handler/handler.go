package handler

import (
	"context"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	corelog "github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"
)

func init() {
	fieldLogger = corelog.NewFieldLogger().WithPackage("sdk.agent.handler")
}

// Handler interface used by the EventListener to process events.
type Handler interface {
	// Handle receives the type of the event context, event metadata and the API Server resource, if it exists.
	Handle(ctx context.Context, eventMetadata *proto.EventMeta, resource *v1.ResourceInstance) error
	ShouldHandle(context.Context, *proto.Event) bool
	// Kinds returns the resource Kinds this Handler cares about, used by EventListener to index
	// Handlers by Kind so events are only dispatched to Handlers whose Kinds() include them. A
	// nil/empty return means the Handler is never dispatched to.
	Kinds() []string
}

// This type is used for values added to context
type ctxKey int

// The key used for the logger in the context
const (
	ctxLogger ctxKey = iota
	ctxAction
)

var fieldLogger corelog.FieldLogger

// logger constants
const (
	handlerField    = "handler"
	sequenceIDField = "sequence"
	actionField     = "action"
	typeField       = "resource"
	nameField       = "name"
)

// client is an interface that is implemented by the ServiceClient in apic/client.go.
type client interface {
	GetResource(url string) (*v1.ResourceInstance, error)
	UpdateResourceFinalizer(ri *v1.ResourceInstance, finalizer, description string, addAction bool) (*v1.ResourceInstance, error)
	CreateSubResource(rm v1.ResourceMeta, subs map[string]interface{}) error
	UpdateResourceInstance(ri v1.Interface) (*v1.ResourceInstance, error)
	DeleteResourceInstance(ri v1.Interface) error
}

func isStatusFound(rs *v1.ResourceStatus) bool {
	if rs == nil || rs.Level == "" {
		return false
	}
	return true
}

// NewEventFromResource builds a synthetic *proto.Event from an already-fetched resource, for
// callers (e.g. StreamWatchProxyHandler, discoveryCache) that only have a *v1.ResourceInstance
// and need to invoke Handler.ShouldHandle before Handle.
func NewEventFromResource(action proto.Event_Type, eventMetadata *proto.EventMeta, resource *v1.ResourceInstance) *proto.Event {
	payload := &proto.ResourceInstance{
		Metadata: &proto.Metadata{},
	}
	if resource != nil {
		payload.Group = resource.Group
		payload.Kind = resource.Kind
		payload.Name = resource.Name
		payload.Attributes = resource.Attributes
		payload.Metadata.Id = resource.Metadata.ID
		payload.Metadata.SelfLink = resource.Metadata.SelfLink
		payload.Metadata.Scope = &proto.Metadata_ScopeKind{
			Id:       resource.Metadata.Scope.ID,
			Kind:     resource.Metadata.Scope.Kind,
			Name:     resource.Metadata.Scope.Name,
			SelfLink: resource.Metadata.Scope.SelfLink,
		}
	}
	return &proto.Event{
		Type:     action,
		Metadata: eventMetadata,
		Payload:  payload,
	}
}

// NewEventContext - create a context for the new event
func NewEventContext(action proto.Event_Type, eventMetadata *proto.EventMeta, kind, name string) context.Context {
	logger := fieldLogger.WithFields(
		logrus.Fields{
			actionField: action.String(),
			typeField:   kind,
			nameField:   name,
		},
	)
	if eventMetadata != nil {
		logger = logger.
			WithField(sequenceIDField, eventMetadata.SequenceID)
	}
	return setActionInContext(setLoggerInContext(context.Background(), logger), action)
}

func setLoggerInContext(ctx context.Context, logger corelog.FieldLogger) context.Context {
	return context.WithValue(ctx, ctxLogger, logger)
}

func setActionInContext(ctx context.Context, action proto.Event_Type) context.Context {
	return context.WithValue(ctx, ctxAction, action)
}

// getLoggerFromContext - returns the field logger that is part of the context
func getLoggerFromContext(ctx context.Context) corelog.FieldLogger {
	return ctx.Value(ctxLogger).(corelog.FieldLogger)
}

// GetActionFromContext retrieve event type from the context
func GetActionFromContext(ctx context.Context) proto.Event_Type {
	return ctx.Value(ctxAction).(proto.Event_Type)
}
