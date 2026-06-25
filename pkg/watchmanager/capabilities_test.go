package watchmanager

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func Test_SetCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []Capability
		expectCaps   []string
	}{
		{
			name:         "should set capabilities on request",
			capabilities: []Capability{CapabilityPing},
			expectCaps:   []string{"ping"},
		},
		{
			name:         "should skip unsupported capabilities",
			capabilities: []Capability{CapabilityPing, "unsupported"},
			expectCaps:   []string{"ping"},
		},
		{
			name:         "should not modify request when all capabilities are unsupported",
			capabilities: []Capability{"unsupported"},
			expectCaps:   nil,
		},
		{
			name:         "should not modify request when capabilities is empty",
			capabilities: []Capability{},
			expectCaps:   nil,
		},
		{
			name:         "should not modify request when capabilities is nil",
			capabilities: nil,
			expectCaps:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &proto.Request{}
			SetCapabilities(req, tc.capabilities)
			assert.Equal(t, tc.expectCaps, req.Capabilities)
		})
	}
}

func Test_HasCapability(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []string
		check        Capability
		expect       bool
	}{
		{
			name:         "should return true when capability is present",
			capabilities: []string{"ping"},
			check:        CapabilityPing,
			expect:       true,
		},
		{
			name:         "should return false when capability is absent",
			capabilities: []string{"other"},
			check:        CapabilityPing,
			expect:       false,
		},
		{
			name:         "should return false when capabilities list is empty",
			capabilities: []string{},
			check:        CapabilityPing,
			expect:       false,
		},
		{
			name:         "should return false when capabilities is nil",
			capabilities: nil,
			check:        CapabilityPing,
			expect:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &proto.Request{Capabilities: tc.capabilities}
			assert.Equal(t, tc.expect, HasCapability(req, tc.check))
		})
	}
}
