package provisioningwebhook

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

var logger = log.NewFieldLogger().WithComponent("provisioningWebhook").WithPackage("agent")

// Dispatch sends payload as the JSON body of a POST request to the webhook configured in cfg, applying
// whichever auth method is configured, using client to make the call. The agent does not wait for the
// client's webhook to actually finish processing - this only reports (via the returned error and logging)
// whether the HTTP call itself was accepted, not whether provisioning/deprovisioning completed.
func Dispatch(client api.Client, cfg config.ProvisioningWebhookEndpointConfig, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		logger.WithError(err).Error("failed to build provisioning webhook payload")
		return err
	}

	headers := map[string]string{"Content-Type": "application/json"}
	for k, v := range cfg.GetWebhookHeaders() {
		headers[k] = v
	}
	applyAuth(cfg, headers)

	resp, err := client.Send(api.Request{
		Method:  api.POST,
		URL:     cfg.GetURL(),
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		logger.WithError(err).Error("failed to call provisioning webhook")
		return err
	}

	if resp.Code < 200 || resp.Code >= 300 {
		err := fmt.Errorf("provisioning webhook returned status %d", resp.Code)
		logger.WithError(err).Error("provisioning webhook call failed")
		return err
	}

	logger.Trace("dispatched to provisioning webhook")
	return nil
}

func applyAuth(cfg config.ProvisioningWebhookEndpointConfig, headers map[string]string) {
	switch cfg.GetAuthType() {
	case config.ProvisioningWebhookAuthBasic:
		creds := base64.StdEncoding.EncodeToString([]byte(cfg.GetUsername() + ":" + cfg.GetPassword()))
		headers["Authorization"] = "Basic " + creds
	case config.ProvisioningWebhookAuthAPIKey:
		headers[cfg.GetAPIKeyHeader()] = cfg.GetAPIKeyValue()
	case config.ProvisioningWebhookAuthBearer:
		headers["Authorization"] = "Bearer " + cfg.GetSecret()
	}
}
