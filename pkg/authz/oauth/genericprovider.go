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
