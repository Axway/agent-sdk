package oauth

import coreapi "github.com/Axway/agent-sdk/pkg/api"

const (
	genericAuthHeaderPrefix = "Bearer"
)

type genericIDP struct {
}

func (i *genericIDP) getAuthorizationHeaderPrefix() string {
	return genericAuthHeaderPrefix
}

func (i *genericIDP) preProcessClientRequest(clientRequest *clientMetadata) {
	// no op
}

func (i *genericIDP) validateExtraProperties(extraProps map[string]interface{}) error {
	return nil
}

func (i *genericIDP) postProcessClientRegistration(clientRes ClientMetadata, credentialObj interface{}, apiClient coreapi.Client) (map[string]string, error) {
	return nil, nil
}

func (i *genericIDP) postProcessClientUnregister(clientID string, agentDetails map[string]string, credentialObj interface{}, apiClient coreapi.Client) error {
	return nil
}
