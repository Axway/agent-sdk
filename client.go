package apic

import (
	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/pkg/config"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
)

//CatalogCreator - interface
type CatalogCreator interface {
	CreateCatalogItemBodyForAdd(bodyForAdd CatalogItemBodyAddParam) ([]byte, error)
	CreateCatalogItemBodyForUpdate(bodyForUpdate CatalogItemBodyUpdateParam) ([]byte, error)
	AddCatalogItem(addCatalogItem AddCatalogItemParam) (string, error)
	UpdateCatalogItem(updateCatalogItem UpdateCatalogItemParam) (string, error)
	AddCatalogItemImage(addCatalogImage AddCatalogItemImageParam) (string, error)
	CreateAPIServerBodyForAdd(apiID, apiName, stageName string, stageTags []string) ([]byte, error)
	AddAPIServer(apiServerBuffer []byte, agentMode corecfg.AgentMode, apiServerEnv string) (string, error)
	DeployAPI(method string, apiServerBuffer []byte, agentMode corecfg.AgentMode, url string) (string, error)
}

//CatalogItemBodyAddParam -
type CatalogItemBodyAddParam struct {
	NameToPush    string
	URL           string
	TeamID        string
	Description   string
	Version       string
	Swagger       []byte
	Documentation []byte
	StageTags     []string
}

//CatalogItemBodyUpdateParam -
type CatalogItemBodyUpdateParam struct {
	NameToPush  string
	Description string
	TeamID      string
	Version     string
	StageTags   []string
}

//AddCatalogItemParam -
type AddCatalogItemParam struct {
	URL       string
	Buffer    []byte
	AgentMode corecfg.AgentMode
}

//UpdateCatalogItemParam -
type UpdateCatalogItemParam struct {
	URL       string
	Buffer    []byte
	AgentMode corecfg.AgentMode
}

//AddCatalogItemImageParam -
type AddCatalogItemImageParam struct {
	URL              string
	CatalogItemID    string
	AgentMode        corecfg.AgentMode
	Image            string
	ImageContentType string
}

// Client -
type Client struct {
	url            string
	tenantID       string
	teamID         string
	tokenRequester *apicauth.PlatformTokenGetter
}

// New -
func New() *Client {

	cfg := config.GetConfig()

	tokenURL := config.GetConfig().CentralConfig.GetAuthConfig().GetTokenURL()
	aud := config.GetConfig().CentralConfig.GetAuthConfig().GetAudience()
	tenantID := config.GetConfig().CentralConfig.GetTenantID()
	priKey := config.GetConfig().CentralConfig.GetAuthConfig().GetPrivateKey()
	pubKey := config.GetConfig().CentralConfig.GetAuthConfig().GetPublicKey()
	keyPwd := config.GetConfig().CentralConfig.GetAuthConfig().GetKeyPassword()
	clientID := config.GetConfig().CentralConfig.GetAuthConfig().GetClientID()
	authTimeout := config.GetConfig().CentralConfig.GetAuthConfig().GetTimeout()
	tokenRequester = apicauth.NewPlatformTokenGetter(priKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout)

	return &Client{
		url:            cfg.CentralConfig.URL,
		tenantID:       tenantID,
		teamID:         cfg.CentralConfig.TeamID,
		tokenRequester: apicauth.NewPlatformTokenGetter(priKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout),
	}
}
