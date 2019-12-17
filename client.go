package apic

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"

	coreapi "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/api"
	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// consts for auth policy types
const (
	Apikey      = "verify-api-key"
	Passthrough = "pass-through"
)

// ValidPolicies - list of valid auth policies supported by Central.  Add to this list as more policies are supported.
var ValidPolicies = []string{Apikey, Passthrough}

//CatalogCreator - interface
type CatalogCreator interface {
	CreateService(serviceBody ServiceBody) ([]byte, error)
	ExecuteService(service Service) (string, error)
	DeployAPI(service Service)
}

//ServiceBody -
type ServiceBody struct {
	NameToPush       string `json:",omitempty"`
	APIName          string `json:",omitempty"`
	URL              string `json:",omitempty"`
	Stage            string `json:",omitempty"`
	TeamID           string `json:",omitempty"`
	Description      string `json:",omitempty"`
	Version          string `json:",omitempty"`
	AuthPolicy       string `json:",omitempty"`
	Swagger          []byte `json:",omitempty"`
	Documentation    []byte `json:",omitempty"`
	Tags             map[string]interface{}
	AgentMode        corecfg.AgentMode `json:",omitempty"`
	ServiceExecution int               `json:"omitempty"`
}

//Service - Used for both adding and updating of catalog item
type Service struct {
	Method    string            `json:",omitempty"`
	URL       string            `json:",omitempty"`
	Buffer    []byte            `json:",omitempty"`
	AgentMode corecfg.AgentMode `json:",omitempty"`
}

// Client -
type Client struct {
	tokenRequester *apicauth.PlatformTokenGetter
	cfg            corecfg.CentralConfig
	apiClient      *coreapi.Client
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
		apiClient:      coreapi.NewClient(cfg.GetTLSConfig()),
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

var log logrus.FieldLogger = logrus.WithField("package", "apic")

// SetLog sets the logger for the package.
func SetLog(newLog logrus.FieldLogger) {
	log = newLog
	return
}

// DeployAPI -
func (c *Client) DeployAPI(service Service) (string, error) {
	headers, err := c.createHeader()
	if err != nil {
		return "", err
	}

	request := coreapi.Request{
		Method:      service.Method,
		URL:         service.URL,
		QueryParams: nil,
		Headers:     headers,
		Body:        service.Buffer,
	}
	response, err := c.apiClient.Send(request)
	if err != nil {
		return "", err
	}
	if !(response.Code == http.StatusOK || response.Code == http.StatusCreated) {
		logResponseErrors(response.Body)
		return "", errors.New(strconv.Itoa(response.Code))
	}

	return handleResponse(service.AgentMode, response.Body)
}

type apiErrorResponse map[string][]apiError

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func logResponseErrors(body []byte) {
	detail := make(map[string]*json.RawMessage)
	json.Unmarshal(body, &detail)

	for k, v := range detail {
		buffer, _ := v.MarshalJSON()
		log.Debugf("HTTP response %v: %v", k, string(buffer))
	}
}

func handleResponse(agentMode corecfg.AgentMode, body []byte) (string, error) {

	itemID := ""

	// Connected Mode
	if agentMode == corecfg.Connected {
		metadata := gjson.Get(string(body), "metadata").String()
		if metadata != "" {
			itemID = gjson.Get(string(metadata), "id").String()
		}
		// Disconnected Mode
	} else {
		itemID = gjson.Get(string(body), "id").String()
	}

	log.Debugf("HTTP response returning itemID: [%v]", itemID)
	return itemID, nil
}

func (c *Client) createHeader() (map[string]string, error) {
	token, err := c.tokenRequester.GetToken()
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	headers["X-Axway-Tenant-Id"] = c.cfg.GetTenantID()
	headers["Authorization"] = "Bearer " + token
	headers["Content-Type"] = "application/json"
	return headers, nil
}

// QueryAPI -
func (c *Client) QueryAPI(apiName string) string {
	var token string
	request, err := http.NewRequest("GET", c.cfg.GetAPIServerServicesURL()+"/"+apiName, nil)

	if token, err = c.tokenRequester.GetToken(); err != nil {
		log.Error("Could not get token")
	}

	request.Header.Add("X-Axway-Tenant-Id", c.cfg.GetTenantID())
	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Add("Content-Type", "application/json")

	response, _ := http.DefaultClient.Do(request)
	if response.StatusCode == http.StatusNotFound {
		log.Debug("New api found to deploy")
		return apiName
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("Could not validate if api " + apiName + " exists.")
	}

	metadata := gjson.Get(string(body), "metadata").String()
	if metadata != "" {
		return apiName + gjson.Get(string(metadata), "id").String()
	}
	return apiName
}
