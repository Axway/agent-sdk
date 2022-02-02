package apic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	cache2 "github.com/Axway/agent-sdk/pkg/agent/cache"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// constants for auth policy types
const (
	Apikey      = "verify-api-key"
	Passthrough = "pass-through"
	Oauth       = "verify-oauth-token"
)

// other consts
const (
	TeamMapKey = "TeamMap"
)

// ValidPolicies - list of valid auth policies supported by Central.  Add to this list as more policies are supported.
var ValidPolicies = []string{Apikey, Passthrough, Oauth}

// SubscriptionProcessor - callback method type to process subscriptions
type SubscriptionProcessor func(subscription Subscription)

// SubscriptionValidator - callback method type to validate subscription for processing
type SubscriptionValidator func(subscription Subscription) bool

// Client - interface
type Client interface {
	SetTokenGetter(tokenRequester auth.PlatformTokenGetter)
	SetConfig(cfg corecfg.CentralConfig)
	PublishService(serviceBody *ServiceBody) (*v1alpha1.APIService, error)
	RegisterSubscriptionWebhook() error
	RegisterSubscriptionSchema(subscriptionSchema SubscriptionSchema, update bool) error
	UpdateSubscriptionSchema(subscriptionSchema SubscriptionSchema) error
	GetSubscriptionManager() SubscriptionManager
	GetCatalogItemIDForConsumerInstance(instanceID string) (string, error)
	DeleteAPIServiceInstance(name string) error
	DeleteConsumerInstance(name string) error
	DeleteServiceByName(name string) error
	GetConsumerInstanceByID(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error)
	GetConsumerInstancesByExternalAPIID(externalAPIID string) ([]*v1alpha1.ConsumerInstance, error)
	UpdateConsumerInstanceSubscriptionDefinition(externalAPIID, subscriptionDefinitionName string) error
	GetUserEmailAddress(ID string) (string, error)
	GetUserName(ID string) (string, error)
	GetSubscriptionsForCatalogItem(states []string, catalogItemID string) ([]CentralSubscription, error)
	GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (SubscriptionSchema, error)
	UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, subscriptionSchema SubscriptionSchema) error
	GetCatalogItemName(ID string) (string, error)
	ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
	Healthcheck(name string) *hc.Status
	GetAPIRevisions(queryParams map[string]string, stage string) ([]*v1alpha1.APIServiceRevision, error)
	GetAPIServiceRevisions(queryParams map[string]string, URL, stage string) ([]*v1alpha1.APIServiceRevision, error)
	GetAPIServiceInstances(queryParams map[string]string, URL string) ([]*v1alpha1.APIServiceInstance, error)
	GetAPIV1ResourceInstances(queryParams map[string]string, URL string) ([]*apiv1.ResourceInstance, error)
	GetAPIV1ResourceInstancesWithPageSize(queryParams map[string]string, URL string, pageSize int) ([]*apiv1.ResourceInstance, error)
	GetAPIServiceByName(serviceName string) (*v1alpha1.APIService, error)
	GetAPIServiceInstanceByName(serviceInstanceName string) (*v1alpha1.APIServiceInstance, error)
	GetAPIRevisionByName(serviceRevisionName string) (*v1alpha1.APIServiceRevision, error)
	CreateCategory(categoryName string) (*catalog.Category, error)
	GetOrCreateCategory(category string) string
	GetEnvironment() (*v1alpha1.Environment, error)
	GetCentralTeamByName(teamName string) (*definitions.PlatformTeam, error)
	GetTeam(queryParams map[string]string) ([]definitions.PlatformTeam, error)
	GetAccessControlList(aclName string) (*v1alpha1.AccessControlList, error)
	UpdateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error)
	CreateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error)
}

// New creates a new Client
func New(cfg corecfg.CentralConfig, tokenRequester auth.PlatformTokenGetter, caches cache2.Manager) Client {
	serviceClient := &ServiceClient{
		caches: caches,
	}
	serviceClient.SetTokenGetter(tokenRequester)
	serviceClient.subscriptionSchemaCache = cache.New()
	serviceClient.initClient(cfg)

	return serviceClient
}

// getTeamFromCache -
func (c *ServiceClient) getTeamFromCache(teamName string) (string, bool) {
	var team *definitions.PlatformTeam
	if teamName == "" {
		team = c.caches.GetDefaultTeam()
		if team == nil {
			return "", false
		}
		return team.ID, true
	}

	team = c.caches.GetTeamByName(teamName)
	if team == nil {
		return "", false
	}

	return team.ID, true
}

// GetOrCreateCategory - Returns the value on published proxy
func (c *ServiceClient) GetOrCreateCategory(category string) string {
	categoryCache := c.caches.GetCategoryCache()
	if categoryCache == nil {
		log.Errorf("category cache has not been initialized")
		return ""
	}

	categoryInterface, _ := categoryCache.GetBySecondaryKey(category)
	if categoryInterface == nil {
		if !corecfg.IsCategoryAutocreationEnabled() {
			log.Warnf("Category auto creation is disabled: agent is not allowed to create %s category", category)
			return ""
		}

		// create the category and add it to the cache
		newCategory, err := c.CreateCategory(category)
		if err != nil {
			log.Errorf(errors.Wrap(ErrCategoryCreate, err.Error()).FormatError(category).Error())
			return ""
		}
		categoryInterface, _ = newCategory.AsInstance()
		log.Infof("Created new category %s (%s)", newCategory.Title, newCategory.Name)
		categoryCache.SetWithSecondaryKey(newCategory.Name, newCategory.Title, categoryInterface)
	}

	cat, ok := categoryInterface.(*apiv1.ResourceInstance)
	if !ok {
		return ""
	}

	return cat.Name
}

// initClient - config change handler
func (c *ServiceClient) initClient(cfg corecfg.CentralConfig) {
	c.cfg = cfg
	c.apiClient = coreapi.NewClientWithTimeout(cfg.GetTLSConfig(), cfg.GetProxyURL(), cfg.GetClientTimeout())
	c.DefaultSubscriptionSchema = NewSubscriptionSchema(cfg.GetEnvironmentName() + SubscriptionSchemaNameSuffix)

	err := c.setTeamCache()
	if err != nil {
		log.Error(err)
	}

	// set the default webhook if one has been configured
	if cfg.GetSubscriptionConfig() != nil {
		webCfg := cfg.GetSubscriptionConfig().GetSubscriptionApprovalWebhookConfig()
		if webCfg != nil && webCfg.IsConfigured() {
			c.DefaultSubscriptionApprovalWebhook = webCfg
		}

		if c.subscriptionMgr == nil {
			c.subscriptionMgr = newSubscriptionManager(c)
		} else {
			c.subscriptionMgr.OnConfigChange(c)
		}
	}
}

// SetTokenGetter - sets the token getter
func (c *ServiceClient) SetTokenGetter(tokenRequester auth.PlatformTokenGetter) {
	c.tokenRequester = tokenRequester
}

// SetConfig - sets the config and apiClient
func (c *ServiceClient) SetConfig(cfg corecfg.CentralConfig) {
	c.cfg = cfg
	c.apiClient = coreapi.NewClientWithTimeout(cfg.GetTLSConfig(), cfg.GetProxyURL(), cfg.GetClientTimeout())
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

func readResponseErrors(statuscode int, body []byte) string {
	// Return error string only for error status code
	if statuscode < http.StatusBadRequest {
		return ""
	}

	responseErr := &ResponseError{}
	err := json.Unmarshal(body, &responseErr)
	if err != nil || len(responseErr.Errors) == 0 {
		errStr := getHTTPResponseErrorString(statuscode, body)
		log.Tracef("HTTP response error: %v", string(errStr))
		return errStr
	}

	// Get the first error from the API response errors
	errStr := getAPIResponseErrorString(responseErr.Errors[0])
	log.Tracef("HTTP response error: %s", errStr)
	return errStr
}

func getHTTPResponseErrorString(statuscode int, body []byte) string {
	detail := make(map[string]*json.RawMessage)
	json.Unmarshal(body, &detail)
	errorMsg := ""
	for _, v := range detail {
		buffer, _ := v.MarshalJSON()
		errorMsg = string(buffer)
	}

	errStr := "status - " + strconv.Itoa(statuscode)
	if errorMsg != "" {
		errStr += ", detail - " + errorMsg
	}
	return errStr
}

func getAPIResponseErrorString(apiError APIError) string {
	errStr := "status - " + strconv.Itoa(apiError.Status)
	if apiError.Title != "" {
		errStr += ", title - " + apiError.Title
	}
	if apiError.Detail != "" {
		errStr += ", detail - " + apiError.Detail
	}
	return errStr
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

// Healthcheck - verify connection to the platform
func (c *ServiceClient) Healthcheck(_ string) *hc.Status {
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

	_, err = c.GetEnvironment()
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
	// this doesn't make a call to platform every time. Only when the token is close to expiring.
	_, err := c.tokenRequester.GetToken()
	if err != nil {
		return errors.Wrap(ErrAuthenticationCall, err.Error())
	}
	return nil
}

func (c *ServiceClient) setTeamCache() error {
	// passing nil to getTeam will return the full list of teams
	platformTeams, err := c.GetTeam(make(map[string]string))
	if err != nil {
		return err
	}

	teamMap := make(map[string]string)
	for _, team := range platformTeams {
		teamMap[team.Name] = team.ID
	}
	return cache.GetCache().Set(TeamMapKey, teamMap)
}

// GetEnvironment get an environment
func (c *ServiceClient) GetEnvironment() (*v1alpha1.Environment, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, errors.Wrap(ErrAuthenticationCall, err.Error())
	}

	queryParams := map[string]string{}

	// do a request for the environment
	bytes, err := c.sendServerRequest(c.cfg.GetEnvironmentURL(), headers, queryParams)
	if err != nil {
		return nil, err
	}

	// Get env id from apiServerEnvByte
	env := &v1alpha1.Environment{}
	err = json.Unmarshal(bytes, env)
	if err != nil {
		return nil, errors.Wrap(ErrEnvironmentQuery, err.Error())
	}

	// Validate that we actually get an environment ID back within the Metadata
	if env.Metadata.ID == "" {
		return nil, ErrEnvironmentQuery
	}

	return env, nil
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
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}
}

// GetPlatformUserInfo - request the platform user info
func (c *ServiceClient) getPlatformUserInfo(id string) (*definitions.PlatformUserInfo, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	platformURL := fmt.Sprintf("%s/api/v1/user/%s", c.cfg.GetPlatformURL(), id)
	log.Tracef("Platform URL being used to get user information %s", platformURL)

	platformUserBytes, reqErr := c.sendServerRequest(platformURL, headers, make(map[string]string, 0))
	if reqErr != nil {
		if reqErr.(*errors.AgentError).GetErrorCode() == ErrRequestQuery.GetErrorCode() {
			return nil, ErrNoAddressFound.FormatError(id)
		}
		return nil, reqErr
	}

	var platformUserInfo definitions.PlatformUserInfo
	err = json.Unmarshal(platformUserBytes, &platformUserInfo)
	if err != nil {
		return nil, err
	}

	return &platformUserInfo, nil
}

// GetUserEmailAddress - request the user email
func (c *ServiceClient) GetUserEmailAddress(id string) (string, error) {

	platformUserInfo, err := c.getPlatformUserInfo(id)
	if err != nil {
		return "", err
	}

	email := platformUserInfo.Result.Email
	log.Tracef("Platform user email %s", email)

	return email, nil
}

// GetUserName - request the user name
func (c *ServiceClient) GetUserName(id string) (string, error) {
	platformUserInfo, err := c.getPlatformUserInfo(id)
	if err != nil {
		return "", err
	}

	userName := fmt.Sprintf("%s %s", platformUserInfo.Result.Firstname, platformUserInfo.Result.Lastname)

	log.Tracef("Platform user %s", userName)

	return userName, nil
}

// GetCentralTeamByName - returns the team based on team name
func (c *ServiceClient) GetCentralTeamByName(teamName string) (*definitions.PlatformTeam, error) {
	// Query for the default, if no teamName is given
	queryParams := map[string]string{}

	if teamName != "" {
		queryParams = map[string]string{
			"query": fmt.Sprintf("name==\"%s\"", teamName),
		}
	}

	platformTeams, err := c.GetTeam(queryParams)
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

// GetTeam - returns the team ID based on filter
func (c *ServiceClient) GetTeam(filterQueryParams map[string]string) ([]definitions.PlatformTeam, error) {
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

	var platformTeams []definitions.PlatformTeam
	err = json.Unmarshal(response, &platformTeams)
	if err != nil {
		return nil, err
	}

	return platformTeams, nil
}

//GetAccessControlList -
func (c *ServiceClient) GetAccessControlList(aclName string) (*v1alpha1.AccessControlList, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/%s", c.cfg.GetEnvironmentACLsURL(), aclName),
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}

	if response.Code != http.StatusOK {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	var acl *v1alpha1.AccessControlList
	err = json.Unmarshal(response.Body, &acl)
	if err != nil {
		return nil, err
	}

	return acl, err
}

//UpdateAccessControlList -
func (c *ServiceClient) UpdateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error) {
	return c.deployAccessControl(acl, http.MethodPut)
}

//CreateAccessControlList -
func (c *ServiceClient) CreateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error) {
	return c.deployAccessControl(acl, http.MethodPost)
}

func (c *ServiceClient) deployAccessControl(acl *v1alpha1.AccessControlList, method string) (*v1alpha1.AccessControlList, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(*acl)
	if err != nil {
		return nil, err
	}

	url := c.cfg.GetEnvironmentACLsURL()
	if method == http.MethodPut {
		url = fmt.Sprintf("%s/%s", url, acl.Name)
	}

	request := coreapi.Request{
		Method:  method,
		URL:     url,
		Headers: headers,
		Body:    data,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}

	if response.Code != http.StatusCreated && response.Code != http.StatusOK {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	updatedACL := &v1alpha1.AccessControlList{}
	err = json.Unmarshal(response.Body, updatedACL)
	if err != nil {
		return nil, err
	}

	return updatedACL, err
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
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}
}
