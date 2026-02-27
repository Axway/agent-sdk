package agent

import (
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/stretchr/testify/assert"
)

func setupCredCache(expireTime time.Time) {
	cred := management.NewCredential("cred", "env")
	cred.Policies.Expiry = &management.CredentialPoliciesExpiry{
		Timestamp: v1.Time(expireTime),
	}
	ri, _ := cred.AsInstance()

	agent.cacheManager = agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	agent.cacheManager.AddWatchResource(ri)
}

func setupAPICClient(mockResponse []api.MockResponse) {
	client, httpClient := apic.GetTestServiceClient()
	httpClient.SetResponses(mockResponse)
	agent.apicClient = client
}

func TestRegisterCredentialChecker(t *testing.T) {
	setupCredCache(time.Time{})
	setupAPICClient([]api.MockResponse{})
	cfg := createCentralCfg("apicentral.axway.com", envName)
	agent.cfg = cfg
	Initialize(cfg)

	c := registerCredentialChecker()
	assert.NotNil(t, c)

	jobs.UnregisterJob(c.id)
}

func TestCredentialValidatorExecute(t *testing.T) {

	tests := []struct {
		name                string
		expectedState       string
		expectedStateReason string
		expectedStatus      string
		expireTime          time.Time
	}{
		{
			name:                "should update expired credential",
			expectedState:       v1.Inactive,
			expectedStateReason: provisioning.CredExpDetail,
			expectedStatus:      provisioning.Pending.String(),
			expireTime:          time.Now().Add(-1 * time.Hour),
		},
		{
			name:       "should not update credential that has not expired",
			expireTime: time.Now().Add(1 * time.Hour),
		},
		{
			name:       "should not update credential that does not expired",
			expireTime: time.Time{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockClient{
				t:                   t,
				expectedState:       tc.expectedState,
				expectedStateReason: tc.expectedStateReason,
				expectedStatus:      tc.expectedStatus,
			}
			setupCredCache(tc.expireTime)
			c := newCredentialChecker(agent.cacheManager, client)
			c.Execute()
		})
	}
}

type mockClient struct {
	apicClient
	t                   *testing.T
	expectedState       string
	expectedStateReason string
	expectedStatus      string
	updatedCred         bool
}

func (c *mockClient) UpdateResourceInstance(ri v1.Interface) (*v1.ResourceInstance, error) {
	c.updatedCred = true
	inst, _ := ri.AsInstance()
	cred := &management.Credential{}
	cred.FromInstance(inst)
	assert.Equal(c.t, c.expectedState, cred.Spec.State.Name)
	assert.Equal(c.t, c.expectedStateReason, cred.Spec.State.Reason)
	assert.Equal(c.t, c.expectedStatus, cred.Status.Level)
	return nil, nil
}

func (c *mockClient) CreateSubResource(rm v1.ResourceMeta, subs map[string]interface{}) error {
	return nil
}
