package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCacheForExternalAPIName(t *testing.T) {
	var queryString string

	teams := []apic.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	environmentRes := &v1alpha1.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "test",
			Title:    "test",
		},
	}

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			queryString = req.URL.RawQuery
			resp.Write([]byte("response"))
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

	testName := "testexternalname"
	api, err := updateCacheForExternalAPIName(testName)

	assert.Nil(t, err)
	assert.NotNil(t, api)
	assert.Contains(t, queryString, fmt.Sprintf("attributes.%s", apic.AttrExternalAPIName))
	assert.Contains(t, queryString, testName)
}

func TestUpdateCacheForExternalAPIID(t *testing.T) {
	var queryString string
	teams := []apic.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	environmentRes := &v1alpha1.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "test",
			Title:    "test",
		},
	}

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			queryString = req.URL.RawQuery
			resp.Write([]byte("response"))
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

	testID := "testexternalid"
	api, err := updateCacheForExternalAPIID(testID)

	assert.Nil(t, err)
	assert.NotNil(t, api)
	assert.Contains(t, queryString, fmt.Sprintf("attributes.%s", apic.AttrExternalAPIID))
	assert.Contains(t, queryString, testID)
}

func TestUpdateCacheForExternalAPIPrimaryKey(t *testing.T) {
	var queryString string
	teams := []apic.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	environmentRes := &v1alpha1.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "test",
			Title:    "test",
		},
	}

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			queryString = req.URL.RawQuery
			resp.Write([]byte("response"))
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

	testKey := "testprimarykey"
	api, err := updateCacheForExternalAPIPrimaryKey(testKey)

	assert.Nil(t, err)
	assert.NotNil(t, api)
	assert.Contains(t, queryString, fmt.Sprintf("attributes.%s", apic.AttrExternalAPIPrimaryKey))
	assert.Contains(t, queryString, testKey)
}
