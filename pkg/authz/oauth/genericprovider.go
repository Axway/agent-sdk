package oauth

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

func (i *genericIDP) validateExtraProperties(extraProps map[string]any) error {
	return nil
}

func (i *genericIDP) postProcessClientRegistration(clientRes ClientMetadata, extraProps map[string]interface{}, credentialObj interface{}) error {
	return nil
}

func (i *genericIDP) postProcessClientUnregister(clientID string, agentDetails map[string]string, extraProps map[string]interface{}, credentialObj interface{}) error {
	return nil
}
