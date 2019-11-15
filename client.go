package apic

import (
	"io"
	"net/http"

	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
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
	SetHeader(method, url string, body io.Reader) (*http.Request, error)
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
	tokenRequester *apicauth.PlatformTokenGetter
	cfg            corecfg.CentralConfig
}

// New -
func New(cfg corecfg.CentralConfig) *Client {
	tokenURL := cfg.GetAuthConfig().GetTokenURL()
	aud := cfg.GetAuthConfig().GetAudience()
	priKey := cfg.GetAuthConfig().GetPrivateKey()
	pubKey := cfg.GetAuthConfig().GetPublicKey()
	keyPwd := cfg.GetAuthConfig().GetKeyPassword()
	clientID := cfg.GetAuthConfig().GetClientID()
	authTimeout := cfg.GetAuthConfig().GetTimeout()

	return &Client{
		cfg:            cfg,
		tokenRequester: apicauth.NewPlatformTokenGetter(priKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout),
	}
}
