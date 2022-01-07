package agent

import (
	"fmt"
	"github.com/Axway/agent-sdk/pkg/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCacheForExternalAPIName(t *testing.T) {
	var queryString string
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			queryString = req.URL.RawQuery
			resp.Write([]byte("response"))
		}
	}))
	defer s.Close()

	cfg := createCentralCfg(s.URL, "test")
	resetResources()
	err := Initialize(cfg, config.NewAgentFeaturesConfiguration())
	assert.Nil(t, err)

	testName := "testexternalname"
	api, err := updateCacheForExternalAPIName(testName)

	assert.Nil(t, err)
	assert.NotNil(t, api)
	assert.Contains(t, queryString, fmt.Sprintf("attributes.%s", apic.AttrExternalAPIName))
	assert.Contains(t, queryString, testName)
}

func TestUpdateCacheForExternalAPIID(t *testing.T) {
	var queryString string
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			queryString = req.URL.RawQuery
			resp.Write([]byte("response"))
		}
	}))
	defer s.Close()

	cfg := createCentralCfg(s.URL, "test")
	resetResources()
	err := Initialize(cfg, config.NewAgentFeaturesConfiguration())
	assert.Nil(t, err)

	testID := "testexternalid"
	api, err := updateCacheForExternalAPIID(testID)

	assert.Nil(t, err)
	assert.NotNil(t, api)
	assert.Contains(t, queryString, fmt.Sprintf("attributes.%s", apic.AttrExternalAPIID))
	assert.Contains(t, queryString, testID)
}

func TestUpdateCacheForExternalAPIPrimaryKey(t *testing.T) {
	var queryString string
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			queryString = req.URL.RawQuery
			resp.Write([]byte("response"))
		}
	}))
	defer s.Close()

	cfg := createCentralCfg(s.URL, "test")
	resetResources()
	err := Initialize(cfg, config.NewAgentFeaturesConfiguration())
	assert.Nil(t, err)

	testKey := "testprimarykey"
	api, err := updateCacheForExternalAPIPrimaryKey(testKey)

	assert.Nil(t, err)
	assert.NotNil(t, api)
	assert.Contains(t, queryString, fmt.Sprintf("attributes.%s", apic.AttrExternalAPIPrimaryKey))
	assert.Contains(t, queryString, testKey)
}
