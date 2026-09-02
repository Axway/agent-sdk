package handler

import (
	"context"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

type customHandler struct {
	err error
}

func (c *customHandler) Handle(_ context.Context, _ *proto.EventMeta, _ *v1.ResourceInstance) error {
	return c.err
}

func (c *customHandler) ShouldHandle(_ context.Context, _ *proto.Event) bool {
	return true
}

// GetAPIServerFields makes customHandler implement RequiredFieldsHandler, so tests can prove the
// proxy no longer consults it.
func (c *customHandler) GetAPIServerFields(_ context.Context, _ *proto.Event) []string {
	return []string{"field"}
}

func TestStreamWatchProxyHandler_RegisterTargetHandler(t *testing.T) {
	tests := []struct {
		name     string
		handlers []*customHandler
	}{
		{
			name:     "no handlers registered for a name",
			handlers: nil,
		},
		{
			name: "a single handler registered for a name",
			handlers: []*customHandler{
				{},
			},
		},
		{
			name: "multiple handlers registered for the same name are appended, not overwritten",
			handlers: []*customHandler{
				{},
				{},
				{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proxy := NewStreamWatchProxyHandler()
			for _, h := range tc.handlers {
				proxy.RegisterTargetHandler("Kind", h)
			}

			got := proxy.GetHandlers()["Kind"]
			assert.Len(t, got, len(tc.handlers))
			for i, h := range tc.handlers {
				assert.Same(t, h, got[i], "handlers must be returned in registration order")
			}
		})
	}
}
