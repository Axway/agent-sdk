package apic

import (
	"crypto/tls"
	"net/http"
	"time"

	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	"git.ecd.axway.int/apigov/v7_discovery_agent/pkg/api"
	"git.ecd.axway.int/apigov/v7_discovery_agent/pkg/config"
)

//CatalogCreator - interface
type CatalogCreator interface {
	CreateCatalogItemBodyForAdd(bodyForAdd CatalogItemBodyAddParam) ([]byte, error)
	CreateCatalogItemBodyForUpdate(bodyForUpdate CatalogItemBodyUpdateParam) ([]byte, error)
	AddCatalogItem(addCatalogItem AddCatalogItemParam) (string, error)
	UpdateCatalogItem(updateCatalogItem UpdateCatalogItemParam) (string, error)
	AddCatalogItemImage(addCatalogImage AddCatalogItemImageParam) (string, error)
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
	url      string
	tenantID string
	teamID   string

	tokenRequester *apicauth.PlatformTokenGetter
	apiClient      *api.Client
}

// New -
func New() *Client {

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: time.Second * 10,
	}

	cfg := config.GetConfig()

	tokenURL := cfg.AuthConfig.URL
	aud := cfg.AuthConfig.Audience
	priKey := cfg.AuthConfig.PrivateKeyPath
	pubKey := cfg.AuthConfig.PublicKeyPath
	keyPwd := cfg.AuthConfig.KeyPassword
	clientID := cfg.AuthConfig.ClientID
	tenantID := cfg.AuthConfig.TenantID
	authTimeout := cfg.AuthConfig.Timeout

	return &Client{
		url:      cfg.CentralConfig.URL,
		tenantID: tenantID,
		teamID:   cfg.CentralConfig.TeamID,

		apiClient:      api.NewClient(httpClient),
		tokenRequester: apicauth.NewPlatformTokenGetter(priKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout),
	}
}
