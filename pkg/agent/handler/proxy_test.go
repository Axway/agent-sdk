package handler

import (
	"context"
	"fmt"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

type customHandler struct {
	err  error
	kind string
}

func (c *customHandler) Handle(_ context.Context, _ *proto.EventMeta, _ *v1.ResourceInstance) error {
	return c.err
}

func (c *customHandler) ShouldHandle(_ context.Context, _ *proto.Event) bool {
	return true
}

func TestProxyHandler(t *testing.T) {
	tests := []struct {
		name     string
		handlers []*customHandler
		event    proto.Event_Type
		hasError bool
	}{
		{
			name:     "should not register any handlers, and return nil when Handle is called",
			event:    proto.Event_UPDATED,
			handlers: nil,
			hasError: false,
		},
		{
			name:  "should register a handler and return nil when Handle is called",
			event: proto.Event_CREATED,
			handlers: []*customHandler{
				&customHandler{},
			},
			hasError: false,
		},
		{
			name:  "should register two handlers and return nil when Handle is called",
			event: proto.Event_CREATED,
			handlers: []*customHandler{
				&customHandler{},
				&customHandler{},
			},
			hasError: false,
		},
		{
			name:  "should register a handler and return an error when Handle is called",
			event: proto.Event_CREATED,
			handlers: []*customHandler{
				&customHandler{err: fmt.Errorf("error")},
			},
			hasError: true,
		},
		{
			name:  "should register two handlers and return an error when Handle is called",
			event: proto.Event_CREATED,
			handlers: []*customHandler{
				&customHandler{},
				&customHandler{err: fmt.Errorf("error")},
			},
			hasError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ri := &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
				},
			}

			proxy := NewStreamWatchProxyHandler()

			for _, h := range tc.handlers {
				proxy.RegisterTargetHandler(h.kind, h)
			}

			err := proxy.Handle(NewEventContext(tc.event, nil, ri.Kind, ri.Name), nil, ri)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}

			for i := range tc.handlers {
				proxy.UnregisterTargetHandler(fmt.Sprintf("%d", i))
			}
		})
	}
}
