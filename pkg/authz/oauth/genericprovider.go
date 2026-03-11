package oauth

import (
	coreapi "github.com/Axway/agent-sdk/pkg/api"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

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

func (i *genericIDP) postProcessClientRegistration(clientRes ClientMetadata, idp corecfg.IDPConfig, apiClient coreapi.Client) (map[string]string, error) {
	return nil, nil
}

func (i *genericIDP) postProcessClientUnregister(clientID string, agentDetails map[string]string, idp corecfg.IDPConfig, apiClient coreapi.Client) error {
	return nil
}
