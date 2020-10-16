package apic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"
	hc "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/healthcheck"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
	"git.ecd.axway.org/apigov/service-mesh-agent/pkg/apicauth"
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
	PublishService(serviceBody ServiceBody) (*v1alpha1.APIService, error)
	RegisterSubscriptionWebhook() error
	RegisterSubscriptionSchema(subscriptionSchema SubscriptionSchema) error
	UpdateSubscriptionSchema(subscriptionSchema SubscriptionSchema) error
	GetSubscriptionManager() SubscriptionManager
	GetCatalogItemIDForConsumerInstance(instanceID string) (string, error)
	DeleteConsumerInstance(instanceName string) error
	GetConsumerInstanceByID(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error)
	GetUserEmailAddress(ID string) (string, error)
	GetSubscriptionsForCatalogItem(states []string, catalogItemID string) ([]CentralSubscription, error)
	GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (SubscriptionSchema, error)
	UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, subscriptionSchema SubscriptionSchema) error
	GetCatalogItemName(ID string) (string, error)
	ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
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

	// set the default webhook if one has been configured
	webCfg := cfg.GetSubscriptionConfig().GetSubscriptionApprovalWebhookConfig()
	if webCfg != nil && webCfg.IsConfigured() {
		serviceClient.DefaultSubscriptionApprovalWebhook = webCfg
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

// SetSubscriptionManager -
func (c *ServiceClient) SetSubscriptionManager(mgr SubscriptionManager) {
	c.subscriptionMgr = mgr
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

func (c *ServiceClient) checkAPIServerHealth() error {

	headers, err := c.createHeader()
	if err != nil {
		return errors.Wrap(ErrAuthenticationCall, err.Error())
	}

	apiEnvironment, err := c.getEnvironment(headers)
	if err != nil || apiEnvironment == nil {
		return err
	}

	if c.cfg.GetEnvironmentID() == "" {
		// need to save this ID for the traceability agent for later
		c.cfg.SetEnvironmentID(apiEnvironment.Metadata.ID)

		err = c.updateEnvironmentStatus(apiEnvironment)
		if err != nil {
			return err
		}
	}

	if c.cfg.GetTeamID() == "" {
		// Validate if team exists
		team, err := c.getCentralTeam(c.cfg.GetTeamName())
		if err != nil {
			return err
		}
		// Set the team Id
		c.cfg.SetTeamID(team.ID)
	}
	return nil
}

func (c *ServiceClient) updateEnvironmentStatus(apiEnvironment *v1alpha1.Environment) error {
	attribute := "x-axway-agent"
	// check to see if x-axway-agent has already been set
	if _, found := apiEnvironment.Attributes[attribute]; found {
		log.Debugf("Environment attribute: %s is already set.", attribute)
		return nil
	}
	apiEnvironment.Attributes[attribute] = "true"

	buffer, err := json.Marshal(apiEnvironment)
	if err != nil {
		return nil
	}
	_, err = c.apiServiceDeployAPI(http.MethodPut, c.cfg.GetEnvironmentURL(), buffer)

	if err != nil {
		return err
	}
	log.Debugf("Updated environment attribute: %s to true.", attribute)
	return nil
}

func (c *ServiceClient) getEnvironment(headers map[string]string) (*v1alpha1.Environment, error) {
	queryParams := map[string]string{}

	// do a request for the environment
	apiEnvByte, err := c.sendServerRequest(c.cfg.GetEnvironmentURL(), headers, queryParams)
	if err != nil {
		return nil, err
	}

	// Get env id from apiServerEnvByte
	var apiEnvironment v1alpha1.Environment
	err = json.Unmarshal(apiEnvByte, &apiEnvironment)
	if err != nil {
		return nil, errors.Wrap(ErrEnvironmentQuery, err.Error())
	}

	// Validate that we actually get an environment ID back within the Metadata
	if apiEnvironment.Metadata.ID == "" {
		return nil, ErrEnvironmentQuery
	}

	return &apiEnvironment, nil
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

// getCentralTeam - returns the team based on team name
func (c *ServiceClient) getCentralTeam(teamName string) (*PlatformTeam, error) {
	// Query for the default, if no teamName is given
	queryParams := map[string]string{}

	if teamName != "" {
		queryParams = map[string]string{
			"query": fmt.Sprintf("name==\"%s\"", teamName),
		}
	}
	platformTeams, err := c.getTeam(queryParams)
	if err != nil {
		return nil, err
	}

	if len(platformTeams) == 0 {
		return nil, ErrTeamNotFound.FormatError(teamName)
	}

	team := platformTeams[0]
	if teamName == "" {
		// Loop through to find the default team
		for i, platformTeam := range platformTeams {
			if platformTeam.Default {
				// Found the default, set as the team var and break
				team = platformTeams[i]
				break
			}
		}
	}

	return &team, nil
}

// getTeam - returns the team ID based on filter
func (c *ServiceClient) getTeam(filterQueryParams map[string]string) ([]PlatformTeam, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	// Get the teams using Client registry service instead of from platform.
	// Platform teams API require access and DOSA account will not have the access
	platformURL := fmt.Sprintf("%s/api/v1/platformTeams", c.cfg.GetURL())

	response, reqErr := c.sendServerRequest(platformURL, headers, filterQueryParams)
	if reqErr != nil {
		return nil, reqErr
	}

	var platformTeams []PlatformTeam
	err = json.Unmarshal(response, &platformTeams)
	if err != nil {
		return nil, err
	}

	return platformTeams, nil
}

// ExecuteAPI - execute the api
func (c *ServiceClient) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:      method,
		URL:         url,
		QueryParams: queryParam,
		Headers:     headers,
		Body:        buffer,
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
