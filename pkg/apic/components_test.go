package apic

import (
	"testing"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestGetComponent(t *testing.T) {
	cases := []struct {
		name                  string
		dataplaneType         string
		agentResourceKind     string
		expectedComponentName string
		expectErr             bool
	}{
		{
			name:                  "GitLab Discovery Agent",
			dataplaneType:         GitLab.String(),
			agentResourceKind:     management.DiscoveryAgentGVK().Kind,
			expectedComponentName: "gitlab-discovery-agent",
		},
		{
			name:              "GitLab Traceability Agent",
			dataplaneType:     GitLab.String(),
			agentResourceKind: management.TraceabilityAgentGVK().Kind,
			expectErr:         true,
		},
		{
			name:              "Dummy Dataplane Type",
			dataplaneType:     "dummy",
			agentResourceKind: management.DiscoveryAgentGVK().Kind,
			expectErr:         true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			componentName, err := GetComponent(tc.dataplaneType, tc.agentResourceKind)
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedComponentName, componentName)
		})
	}
}
