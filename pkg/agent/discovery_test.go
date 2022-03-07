package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/mock"

	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/stretchr/testify/assert"
)

func TestDiscoveryCache(t *testing.T) {
	dcj := newDiscoveryCache(nil, true, &sync.Mutex{})
	dcj.getHCStatus = func(_ string) hc.StatusLevel {
		return hc.OK
	}
	attributeKey := "Attr1"
	attributeValue := "testValue"
	emptyAPISvc := []v1.ResourceInstance{}
	apiSvc1 := v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceGVK(),
			Name:             "testAPIService1",
			Attributes: map[string]string{
				definitions.AttrExternalAPIID:         "1111",
				definitions.AttrExternalAPIPrimaryKey: "1234",
				definitions.AttrExternalAPIName:       "NAME",
				attributeKey:                          attributeValue,
			},
		},
	}
	apiSvc2 := v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceGVK(),
			Name:             "testAPIService2",
			Attributes: map[string]string{
				definitions.AttrExternalAPIID: "2222",
			},
		},
	}
	var serverAPISvcResponse []v1.ResourceInstance
	environmentRes := &v1alpha1.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "test",
			Title:    "test",
		},
	}
	teams := []definitions.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			buf, _ := json.Marshal(serverAPISvcResponse)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test") {
			buf, _ := json.Marshal(environmentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/api/v1/platformTeams") {
			buf, _ := json.Marshal(teams)
			resp.Write(buf)
			return
		}
	}))
	defer s.Close()

	cfg := createCentralCfg(s.URL, "test")
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)

	assert.True(t, dcj.Ready())
	assert.Nil(t, dcj.Status())

	serverAPISvcResponse = emptyAPISvc
	dcj.updateAPICache()
	assert.Equal(t, 0, len(agent.cacheManager.GetAPIServiceKeys()))
	assert.False(t, IsAPIPublishedByID("1111"))
	assert.False(t, IsAPIPublishedByID("2222"))

	serverAPISvcResponse = []v1.ResourceInstance{apiSvc1}
	dcj.updateAPICache()
	assert.Equal(t, 1, len(agent.cacheManager.GetAPIServiceKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.False(t, IsAPIPublishedByID("2222"))
	assert.Equal(t, "1111", GetAttributeOnPublishedAPIByID("1111", definitions.AttrExternalAPIID))
	assert.Equal(t, "", GetAttributeOnPublishedAPIByID("2222", definitions.AttrExternalAPIID))
	assert.Equal(t, attributeValue, GetAttributeOnPublishedAPIByPrimaryKey("1234", attributeKey))
	assert.Equal(t, attributeValue, GetAttributeOnPublishedAPIByName("NAME", attributeKey))

	apicClient := agent.apicClient
	var apiSvc v1alpha1.APIService
	apiSvc.FromInstance(&apiSvc2)
	agent.apicClient = &mock.Client{
		PublishServiceMock: func(serviceBody *apic.ServiceBody) (*v1alpha1.APIService, error) {
			return &apiSvc, nil
		},
	}
	StartAgentStatusUpdate()
	PublishAPI(apic.ServiceBody{})
	agent.apicClient = apicClient
	assert.Equal(t, 2, len(agent.cacheManager.GetAPIServiceKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.True(t, IsAPIPublishedByID("2222"))

	serverAPISvcResponse = []v1.ResourceInstance{apiSvc1}
	dcj.updateAPICache()
	assert.Equal(t, 1, len(agent.cacheManager.GetAPIServiceKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.True(t, IsAPIPublishedByPrimaryKey("1234"))
	assert.False(t, IsAPIPublishedByID("2222"))
}
