package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func createAWSDiscoveryAgentRes(id, name, dataplane, filter string) *v1.ResourceInstance {
	res := &v1alpha1.AWSDiscoveryAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.AwsDiscoveryAgentSpec{
			Dataplane: dataplane,
			Config: v1alpha1.DiscoveryAgentSpecConfig{
				Filter: filter,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createAWSTraceabilityAgentRes(id, name, dataplane string, processHeaders bool) *v1.ResourceInstance {
	res := &v1alpha1.AWSTraceabilityAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.AwsTraceabilityAgentSpec{
			Dataplane: dataplane,
			Config: v1alpha1.TraceabilityAgentSpecConfig{
				ProcessHeaders: processHeaders,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createAWSDataplaneRes(id, name, region, queueName, logGroup string) *v1.ResourceInstance {
	res := &v1alpha1.AWSDataplane{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.AwsDataplaneSpec{
			Region:                   region,
			ResourceChangeEventQueue: queueName,
			TransactionLogGroup:      logGroup,
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func TestAWSAgentInitialize(t *testing.T) {
	var awsDataplaneRes, awsDiscoveryAgentRes, awsTraceabilitAgentRes *v1.ResourceInstance
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments") {
			if strings.Contains(req.RequestURI, "aws/awsdataplanes/aws-dataplane") {
				buf, _ := json.Marshal(awsDataplaneRes)
				resp.Write(buf)
			}
			if strings.Contains(req.RequestURI, "/aws/awsdiscoveryagents/aws-discovery") {
				buf, _ := json.Marshal(awsDiscoveryAgentRes)
				resp.Write(buf)
			}
			if strings.Contains(req.RequestURI, "aws/awstraceabilityagents/aws-traceability") {
				buf, _ := json.Marshal(awsTraceabilitAgentRes)
				resp.Write(buf)
			}
		}
	}))

	defer s.Close()

	cfg := createCentralCfg(s.URL, "aws")
	// Test with no agent name - config to be validate successfully as no calls made to get agent and dataplane resource
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)
	da := GetAgentResource()
	dp := GetDataplaneResource()
	assert.Nil(t, da)
	assert.Nil(t, dp)

	awsDataplaneRes = createAWSDataplaneRes("111", "aws-dataplane", "region", "queueName", "logGrouName")
	awsDiscoveryAgentRes = createAWSDiscoveryAgentRes("111", "aws-discovery", "aws-dataplane", "")
	awsTraceabilitAgentRes = createAWSTraceabilityAgentRes("111", "aws-traceability", "aws-dataplane", false)

	AgentResourceType = v1alpha1.AWSDiscoveryAgentResource
	cfg.AgentName = "aws-discovery"
	resetResources()
	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	dp = GetDataplaneResource()
	assertResource(t, dp, awsDataplaneRes)
	assertResource(t, da, awsDiscoveryAgentRes)

	AgentResourceType = v1alpha1.AWSTraceabilityAgentResource
	cfg.AgentName = "aws-traceability"
	resetResources()
	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	dp = GetDataplaneResource()
	assertResource(t, dp, awsDataplaneRes)
	assertResource(t, da, awsTraceabilitAgentRes)
}
