package apic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/pkg/config"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	"github.com/aws/aws-sdk-go/aws"
	apigw "github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/tidwall/gjson"
)

//CatalogCreator - interface
type CatalogCreator interface {
	CreateCatalogItemBody(bodyForAdd CatalogItemBodyParam) ([]byte, error)
	ProcessCatalogItem(catalogItem CatalogItemParam) (string, error)
	CreateAPIServerBodyForAdd(apiID, apiName, stageName string, tags map[string]interface{}) ([]byte, error)
	AddAPIServer(apiServerBuffer []byte, agentMode corecfg.AgentMode, apiServerEnv string) (string, error)
	DeployAPI(method string, apiServerBuffer []byte, agentMode corecfg.AgentMode, url string) (string, error)
	SetHeader(method, url string, body io.Reader) (*http.Request, error)
}

//CatalogItemBodyParam -
type CatalogItemBodyParam struct {
	NameToPush    string
	URL           string `json:",omitempty"`
	TeamID        string
	Description   string
	Version       string
	AuthPolicy    string `json:",omitempty"`
	Swagger       []byte `json:",omitempty"`
	Documentation []byte `json:",omitempty"`
	Tags          map[string]interface{}
}

//CatalogItemParam - Used for both adding and updating of catalog item
type CatalogItemParam struct {
	Method           string
	URL              string
	Buffer           []byte
	AgentMode        corecfg.AgentMode
	Image            string `json:",omitempty"`
	ImageContentType string `json:",omitempty"`
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

// MapToStringArray -
func (c *Client) MapToStringArray(m map[string]interface{}) []string {
	strArr := []string{}

	for key, val := range m {
		v := val.(*string)
		if *v == "" {
			strArr = append(strArr, key)
		} else {
			strArr = append(strArr, key+"_"+*v)
		}
	}
	return strArr
}

var methods = [5]string{"get", "post", "put", "patch", "delete"} // RestAPI methods

const (
	apikey      = "verify-api-key"
	passthrough = "pass-through"
)

func (c *Client) determineAuthPolicyFromSwagger(swagger *[]byte) string {
	// Traverse the swagger looking for any route that has security set
	// return the security of the first route, if none- found return passthrough
	var authPolicy = passthrough

	gjson.GetBytes(*swagger, "paths").ForEach(func(_, pathObj gjson.Result) bool {
		for _, method := range methods {
			if pathObj.Get(fmt.Sprint(method, ".security.#.api_key")).Exists() {
				authPolicy = apikey
				return false
			}
		}
		return authPolicy == passthrough // Return from path loop anonymous func, true = go to next item
	})

	return authPolicy
}

// build up struct for catalog item body
func (c *Client) buildCatalogItemBody(restAPIID, stageName string, restAPI *apigw.RestApi, exportOut *apigw.GetExportOutput, tags map[string]interface{}) CatalogItemBodyParam {
	// Build up catalog body
	region := config.GetConfig().AWSConfig.GetRegion()
	nameToPush := fmt.Sprintf("%v (Stage: %v)", aws.StringValue(restAPI.Name), stageName)
	url := "https://" + restAPIID + ".execute-api." + region + ".amazonaws.com/" + stageName
	teamID := config.GetConfig().CentralConfig.GetTeamID()
	description := "API From AWS APIGateway (RestApiId: " + restAPIID + ", StageName: " + stageName + ")"
	version := "1.0.0"
	authPolicy := c.determineAuthPolicyFromSwagger(&exportOut.Body)
	desc := gjson.Get(string(exportOut.Body), "info.description")
	documentation := desc.Str
	if documentation == "" {
		documentation = "API imported from AWS APIGateway"
	}
	docBytes, _ := json.Marshal(documentation)

	return CatalogItemBodyParam{
		NameToPush:    nameToPush,
		URL:           url,
		TeamID:        teamID,
		Description:   description,
		Version:       version,
		AuthPolicy:    authPolicy,
		Swagger:       exportOut.Body,
		Documentation: docBytes,
		Tags:          tags,
	}

}

func (c *Client) buildCatalogItemBodyForUpdate(restAPIID, stageName string, restAPI *apigw.RestApi, tags map[string]interface{}) CatalogItemBodyParam {
	nameToPush := fmt.Sprintf("%v (Stage: %v)", aws.StringValue(restAPI.Name), stageName)
	teamID := config.GetConfig().CentralConfig.GetTeamID()
	description := "API From AWS APIGateway Updated (RestApiId: " + restAPIID + ", StageName: " + stageName + ")"
	version := "1.0.1"
	return CatalogItemBodyParam{
		NameToPush:  nameToPush,
		Description: description,
		TeamID:      teamID,
		Version:     version,
		Tags:        tags,
	}
}

// build up struct for catalog item for add and update
func (c *Client) buildCatalogItem(method, apicURL string, catalogBuffer []byte) CatalogItemParam {
	return CatalogItemParam{
		Method:    method,
		URL:       apicURL,
		Buffer:    catalogBuffer,
		AgentMode: config.GetConfig().CentralConfig.GetAgentMode(),
	}
}
