package apic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
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
	GetSubscriptionManager() SubscriptionManager
	DeleteCatalogItem(itemID string) error
	GetConsumerInstanceForCatalogItem(catalogID string) (*APIServer, error)
	DeleteConsumerInstance(instanceName string) error
	GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]CentralSubscription, error)
	DoesCatalogItemForServiceHaveActiveSubscriptions(itemID string) (bool, error)
	InitiateUnsubscribeCatalogItem(catalogItemID string) (int, error)
	UnsubscribeCatalogItem(catalogItemID string) (int, error)
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
		DefaultSubscriptionSchema: NewSubscriptionSchema(),
	}
	serviceClient.subscriptionMgr = newSubscriptionManager(serviceClient)

	hc.RegisterHealthcheck("AMPLIFY Central", "central", serviceClient.healthcheck)
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
	// Check that we can reach the API Manager catalog
	// only concerned if mode: PublishToCatalog
	if c.cfg.IsPublishToCatalogMode() && c.cfg.GetAgentType() != corecfg.TraceabilityAgent {
		err := c.checkCatalogHealth()
		if err != nil {
			s = hc.Status{
				Result:  hc.FAIL,
				Details: err.Error(),
			}
		}
		// Return our response
		return &s
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
		return fmt.Errorf("error trying to get platform token: %s. Check AMPLIFY Central configuration for AUTH_URL, AUTH_REALM, AUTH_CLIENTID, AUTH_PRIVATEKEY, and AUTH_PUBLICKEY", err.Error())
	}
	return nil
}
func (c *ServiceClient) checkCatalogHealth() error {
	// do a request for catalog items
	headers, err := c.createHeader()
	if err != nil {
		return fmt.Errorf("error creating request header. %s", err.Error())
	}

	sendErr := "error sending request to API Manager: %s. Check API Manager for URL and CONNECTION"
	statusErr := "error sending request to API Manager - status code %d. Check API Manager and CONNECTION"

	// do a request for catalog items
	_, err = c.sendServerRequest(c.cfg.GetCatalogItemsURL(), headers, make(map[string]string, 0), sendErr, statusErr)
	return err
}

func (c *ServiceClient) checkAPIServerHealth() error {

	headers, err := c.createHeader()
	if err != nil {
		return fmt.Errorf("error creating request header. %s", err.Error())
	}

	sendErr := "error sending request to API Server: %s. Check AMPLIFY Central configuration for URL and ENVIRONMENT"
	statusErr := "error sending request to API Server - status code %d. Check AMPLIFY Central configuration for ENVIRONMENT"

	// do a request for the environment
	apiServerEnvByte, err := c.sendServerRequest(c.cfg.GetAPIServerEnvironmentURL(), headers, make(map[string]string, 0), sendErr, statusErr)
	if err != nil {
		queryParams := map[string]string{
			"query": fmt.Sprintf("name==\"%s\"", c.cfg.GetEnvironmentName()),
		}
		envListByte, err := c.sendServerRequest(c.cfg.GetEnvironmentURL(), headers, queryParams, sendErr, statusErr)
		if err == nil {
			var envList []EnvironmentSpec
			err := json.Unmarshal(envListByte, &envList)
			if err != nil || len(envList) == 0 {
				return fmt.Errorf("error sending request to API Server. Check AMPLIFY Central configuration for ENVIRONMENT")
			}
			c.cfg.SetEnvironmentID(envList[0].ID)
			return nil
		}
		return err
	}

	// Get end id from apiServerEnvByte
	var apiServerEnv APIServer
	err = json.Unmarshal(apiServerEnvByte, &apiServerEnv)
	if err != nil {
		return fmt.Errorf("error sending request to API Server. Check AMPLIFY Central configuration for ENVIRONMENT")
	}
	c.cfg.SetEnvironmentID(apiServerEnv.Metadata.ID)
	return nil
}

func (c *ServiceClient) sendServerRequest(url string, headers, query map[string]string, sendErr, statusErr string) ([]byte, error) {
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		QueryParams: query,
		Headers:     headers,
	}
	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, fmt.Errorf(sendErr, err.Error())
	}
	if response.Code != http.StatusOK {
		logResponseErrors(response.Body)
		return nil, fmt.Errorf(statusErr, response.Code)
	}
	return response.Body, nil
}

// DeleteCatalogItem -
func (c *ServiceClient) DeleteCatalogItem(itemID string) error {
	// delete doesn't need a service body
	serviceBody := ServiceBody{
		AuthPolicy: Passthrough,
	}
	return c.deleteCatalogItem(itemID, serviceBody)
}

// GetConsumerInstanceForCatalogItem -
func (c *ServiceClient) GetConsumerInstanceForCatalogItem(itemID string) (*APIServer, error) {
	return c.getConsumerInstanceForCatalogItem(itemID)
}

// DeleteConsumerInstance -
func (c *ServiceClient) DeleteConsumerInstance(instanceName string) error {
	return c.deleteConsumerInstance(instanceName)
}

// GetSubscriptionsForCatalogItem -
func (c *ServiceClient) GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]CentralSubscription, error) {
	return c.getSubscriptionsForCatalogItem(states, instanceID)
}

// DoesCatalogItemForServiceHaveActiveSubscriptions -
func (c *ServiceClient) DoesCatalogItemForServiceHaveActiveSubscriptions(instanceID string) (bool, error) {
	return c.doesCatalogItemForServiceHaveActiveSubscriptions(instanceID)
}
