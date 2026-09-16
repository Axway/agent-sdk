package provisioningwebhook

import (
	"errors"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

type fakeClient struct {
	responses []fakeResponse
	calls     int
}

type fakeResponse struct {
	code int
	err  error
}

func (f *fakeClient) Send(_ api.Request) (*api.Response, error) {
	r := f.responses[f.calls]
	f.calls++
	if r.err != nil {
		return nil, r.err
	}
	return &api.Response{Code: r.code}, nil
}

func newCfg(retryCount int) config.ProvisioningWebhookEndpointConfig {
	return &config.ProvisioningWebhookEndpointConfiguration{
		WebhookConfiguration: &config.WebhookConfiguration{URL: "http://webhook.example.com"},
		RetryCount:           retryCount,
	}
}

func TestDispatch_SucceedsWithoutRetry(t *testing.T) {
	client := &fakeClient{responses: []fakeResponse{{code: 200}}}
	err := Dispatch(client, newCfg(2), map[string]string{"a": "b"})
	assert.NoError(t, err)
	assert.Equal(t, 1, client.calls)
}

func TestDispatch_RetriesThenSucceeds(t *testing.T) {
	client := &fakeClient{responses: []fakeResponse{
		{err: errors.New("connection refused")},
		{code: 500},
		{code: 200},
	}}
	err := Dispatch(client, newCfg(2), map[string]string{"a": "b"})
	assert.NoError(t, err)
	assert.Equal(t, 3, client.calls)
}

func TestDispatch_ExhaustsRetriesAndFails(t *testing.T) {
	client := &fakeClient{responses: []fakeResponse{
		{code: 500},
		{code: 500},
		{code: 500},
	}}
	err := Dispatch(client, newCfg(2), map[string]string{"a": "b"})
	assert.Error(t, err)
	assert.Equal(t, 3, client.calls) // initial attempt + 2 retries
}

func TestDispatch_NoRetryConfiguredFailsImmediately(t *testing.T) {
	client := &fakeClient{responses: []fakeResponse{{code: 500}}}
	err := Dispatch(client, newCfg(0), map[string]string{"a": "b"})
	assert.Error(t, err)
	assert.Equal(t, 1, client.calls)
}
