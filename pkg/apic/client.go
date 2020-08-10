package apic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/errors"
	hc "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/healthcheck"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
)

// constants for auth policy types
const (
	Apikey      = "verify-api-key"
	Passthrough = "pass-through"
	Oauth       = "verify-oauth-token"
)

const serverName = "AMPLIFY Central"

// ValidPolicies - list of valid auth policies supported by Central.  Add to this list as more policies are supported.
var ValidPolicies = []string{Apikey, Passthrough, Oauth}

// SubscriptionProcessor - callback method type to process subscriptions
type SubscriptionProcessor func(subscription Subscription)

// SubscriptionValidator - callback method type to validate subscription for processing
type SubscriptionValidator func(subscription Subscription) bool

// Client - interface
type Client interface {
	CreateService(serviceBody ServiceBody) (string, error)
	UpdateService(ID string, serviceBody ServiceBody) (string, error)
	RegisterSubscriptionSchema(subscriptionSchema SubscriptionSchema) error
	UpdateSubscriptionSchema(subscriptionSchema SubscriptionSchema) error
	GetSubscriptionManager() SubscriptionManager
	GetCatalogItemIDForConsumerInstance(instanceID string) (string, error)
	DeleteConsumerInstance(instanceName string) error
	GetConsumerInstanceByInstanceID(consumerInstanceID string) (*APIServer, error)
	GetUserEmailAddress(ID string) (string, error)
	GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]CentralSubscription, error)
	GetCatalogItemName(ID string) (string, error)
}

type tokenGetter interface {
	GetToken() (string, error)
}

type platformTokenGetter struct {
	requester *apicauth.PlatformTokenGetter
}

func (p *platformTokenGetter) GetToken() (string, error) {
	return p.requester.GetToken()
}

// New -
func New(cfg corecfg.CentralConfig) Client {
	tokenURL := cfg.GetAuthConfig().GetTokenURL()
	aud := cfg.GetAuthConfig().GetAudience()
	priKey := cfg.GetAuthConfig().GetPrivateKey()
	pubKey := cfg.GetAuthConfig().GetPublicKey()
	keyPwd := cfg.GetAuthConfig().GetKeyPassword()
	clientID := cfg.GetAuthConfig().GetClientID()
	authTimeout := cfg.GetAuthConfig().GetTimeout()
	platformTokenGetter := &platformTokenGetter{
		requester: apicauth.NewPlatformTokenGetter(priKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout),
	}
	serviceClient := &ServiceClient{
		cfg:                       cfg,
		tokenRequester:            platformTokenGetter,
		apiClient:                 coreapi.NewClient(cfg.GetTLSConfig(), cfg.GetProxyURL()),
		DefaultSubscriptionSchema: NewSubscriptionSchema(cfg.GetEnvironmentName() + SubscriptionSchemaNameSuffix),
	}
	serviceClient.subscriptionMgr = newSubscriptionManager(serviceClient)

	hc.RegisterHealthcheck(serverName, "central", serviceClient.healthcheck)
	return serviceClient
}

// mapToTagsArray -
func (c *ServiceClient) mapToTagsArray(m map[string]interface{}) []string {
	strArr := []string{}

	for key, val := range m {
		var value string
		v, ok := val.(*string)
		if ok {
			value = *v
		} else {
			v, ok := val.(string)
			if ok {
				value = v
			}
		}
		if value == "" {
			strArr = append(strArr, key)
		} else {
			strArr = append(strArr, key+"_"+value)
		}
	}

	// Add any tags from config
	additionalTags := c.cfg.GetTagsToPublish()
	if additionalTags != "" {
		additionalTagsArray := strings.Split(additionalTags, ",")

		for _, tag := range additionalTagsArray {
			strArr = append(strArr, strings.TrimSpace(tag))
		}
	}

	return strArr
}

func isUnitTesting() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}

func logResponseErrors(body []byte) {
	detail := make(map[string]*json.RawMessage)
	json.Unmarshal(body, &detail)

	for k, v := range detail {
		buffer, _ := v.MarshalJSON()
		log.Debugf("HTTP response %v: %v", k, string(buffer))
	}
}

func (c *ServiceClient) createHeader() (map[string]string, error) {
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

// GetSubscriptionManager -
func (c *ServiceClient) GetSubscriptionManager() SubscriptionManager {
	return c.subscriptionMgr
}

func (c *ServiceClient) healthcheck(name string) *hc.Status {
	// Set a default response
	s := hc.Status{
		Result: hc.OK,
	}

	// Check that we can reach the platform
	err := c.checkPlatformHealth()
	if err != nil {
		s = hc.Status{
			Result:  hc.FAIL,
			Details: err.Error(),
		}
	}

	// Check that appropriate settings for the API server are set
	err = c.checkAPIServerHealth()
	if err != nil {
		s = hc.Status{
			Result:  hc.FAIL,
			Details: err.Error(),
		}
	}
	// Return our response
	return &s
}

func (c *ServiceClient) checkPlatformHealth() error {
	_, err := c.tokenRequester.GetToken()
	if err != nil {
		return errors.Wrap(ErrAuthenticationCall, err.Error())
	}
	return nil
}
func (c *ServiceClient) checkCatalogHealth() error {
	// do a request for catalog items
	headers, err := c.createHeader()
	if err != nil {
		return errors.Wrap(ErrAuthenticationCall, err.Error())
	}

	// do a request for catalog items
	_, err = c.sendServerRequest(c.cfg.GetCatalogItemsURL(), headers, make(map[string]string, 0))
	return err
}

func (c *ServiceClient) checkAPIServerHealth() error {

	headers, err := c.createHeader()
	if err != nil {
		return errors.Wrap(ErrAuthenticationCall, err.Error())
	}

	queryParams := map[string]string{"fields": "metadata"}

	// do a request for the environment
	apiServerEnvByte, err := c.sendServerRequest(c.cfg.GetAPIServerEnvironmentURL(), headers, queryParams)
	if err != nil {
		queryParams := map[string]string{
			"query": fmt.Sprintf("name==\"%s\"", c.cfg.GetEnvironmentName()),
		}

		// if the environment wasn't found above, check for it here
		envListByte, err := c.sendServerRequest(c.cfg.GetEnvironmentURL(), headers, queryParams)
		if err == nil {
			var envList []EnvironmentSpec
			err := json.Unmarshal(envListByte, &envList)
			if err != nil || len(envList) == 0 {
				return ErrEnvironmentQuery
			}
			c.cfg.SetEnvironmentID(envList[0].ID)
			return nil
		}
		return err
	}

	// Get env id from apiServerEnvByte
	var apiServerEnv APIServer
	err = json.Unmarshal(apiServerEnvByte, &apiServerEnv)
	if err != nil {
		return errors.Wrap(ErrEnvironmentQuery, err.Error())
	}

	// Validate that we actually get an environment ID back within the Metadata
	if apiServerEnv.Metadata == nil {
		return ErrEnvironmentQuery
	}

	if apiServerEnv.Metadata.ID == "" {
		return ErrEnvironmentQuery
	}

	// need to save this ID for the traceability agent for later
	c.cfg.SetEnvironmentID(apiServerEnv.Metadata.ID)
	return nil
}

func (c *ServiceClient) sendServerRequest(url string, headers, query map[string]string) ([]byte, error) {
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		QueryParams: query,
		Headers:     headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, errors.Wrap(ErrNetwork, err.Error())
	}

	switch response.Code {
	case http.StatusOK:
		return response.Body, nil
	case http.StatusUnauthorized:
		return nil, ErrAuthentication
	default:
		logResponseErrors(response.Body)
		return nil, ErrRequestQuery
	}

}

// GetCatalogItemIDForConsumerInstance -
func (c *ServiceClient) GetCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	return c.getCatalogItemIDForConsumerInstance(instanceID)
}

// DeleteConsumerInstance -
func (c *ServiceClient) DeleteConsumerInstance(instanceName string) error {
	return c.deleteConsumerInstance(instanceName)
}

// GetConsumerInstanceByInstanceID -
func (c *ServiceClient) GetConsumerInstanceByInstanceID(consumerInstanceID string) (*APIServer, error) {
	return c.getConsumerInstanceByInstanceID((consumerInstanceID))
}

// GetSubscriptionsForCatalogItem -
func (c *ServiceClient) GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]CentralSubscription, error) {
	return c.getSubscriptionsForCatalogItem(states, instanceID)
}

// PlatformUserInfo -
type PlatformUserInfo struct {
	Success bool `json:"success"`
	Result  struct {
		ID        string `json:"_id"`
		GUID      string `json:"guid"`
		UserID    int64  `json:"user_id"`
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
		Active    bool   `json:"active"`
		Email     string `json:"email"`
	} `json:"result"`
}

// GetUserEmailAddress - request the user email
func (c *ServiceClient) GetUserEmailAddress(id string) (string, error) {
	headers, err := c.createHeader()
	if err != nil {
		return "", err
	}

	platformURL := fmt.Sprintf("%s/api/v1/user/%s", c.cfg.GetPlatformURL(), id)
	log.Debugf("Platform URL being used to get user information %s", platformURL)

	platformUserBytes, reqErr := c.sendServerRequest(platformURL, headers, make(map[string]string, 0))
	if reqErr != nil {
		if reqErr.(*errors.AgentError).GetErrorCode() == ErrRequestQuery.GetErrorCode() {
			return "", ErrNoAddressFound.FormatError(id)
		}
		return "", reqErr
	}

	// Get the email
	var platformUserInfo PlatformUserInfo
	err = json.Unmarshal(platformUserBytes, &platformUserInfo)
	if err != nil {
		return "", err
	}

	email := platformUserInfo.Result.Email
	log.Debugf("Platform user email %s", email)

	return email, nil
}
