package handler

import (
	"context"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
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
}

func isStatusFound(rs *v1.ResourceStatus) bool {
	if rs == nil || rs.Level == "" {
		return false
	}
	return true
}

func shouldIgnoreSubResourceUpdate(action proto.Event_Type, meta *proto.EventMeta) bool {
	if meta == nil {
		return false
	}
	return action == proto.Event_SUBRESOURCEUPDATED && meta.Subresource != "status"
}

// shouldProcessPending returns true when the resource is pending, and is not in a deleting state
func shouldProcessPending(status, state string) bool {
	return status == prov.Pending.String() && state != v1.ResourceDeleting
}

// shouldProcessDeleting returns true when the resource is in a deleting state and has finalizers
func shouldProcessDeleting(status, state string, finalizerCount int) bool {
	return status == prov.Success.String() && state == v1.ResourceDeleting && finalizerCount > 0
}

func shouldProcessForTrace(status, state string) bool {
	return status == prov.Success.String() && state != v1.ResourceDeleting
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
