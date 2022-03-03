package apic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	cache2 "github.com/Axway/agent-sdk/pkg/agent/cache"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/errors"
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
	PublishService(serviceBody *ServiceBody) (*mv1a.APIService, error)
	RegisterSubscriptionWebhook() error
	RegisterSubscriptionSchema(schema SubscriptionSchema, update bool) error
	UpdateSubscriptionSchema(schema SubscriptionSchema) error
	GetSubscriptionManager() SubscriptionManager
	GetCatalogItemIDForConsumerInstance(instanceID string) (string, error)
	DeleteAPIServiceInstance(name string) error
	DeleteConsumerInstance(name string) error
	DeleteServiceByName(name string) error
	GetConsumerInstanceByID(id string) (*mv1a.ConsumerInstance, error)
	GetConsumerInstancesByExternalAPIID(externalAPIID string) ([]*mv1a.ConsumerInstance, error)
	UpdateConsumerInstanceSubscriptionDefinition(externalAPIID, subscriptionDefinitionName string) error
	GetUserEmailAddress(ID string) (string, error)
	GetUserName(ID string) (string, error)
	GetSubscriptionsForCatalogItem(states []string, catalogItemID string) ([]CentralSubscription, error)
	GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (SubscriptionSchema, error)
	UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, schema SubscriptionSchema) error
	GetCatalogItemName(ID string) (string, error)
	ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
	Healthcheck(name string) *hc.Status
	GetAPIRevisions(query map[string]string, stage string) ([]*mv1a.APIServiceRevision, error)
	GetAPIServiceRevisions(query map[string]string, URL, stage string) ([]*mv1a.APIServiceRevision, error)
	GetAPIServiceInstances(query map[string]string, URL string) ([]*mv1a.APIServiceInstance, error)
	GetAPIV1ResourceInstances(query map[string]string, URL string) ([]*apiv1.ResourceInstance, error)
	GetAPIV1ResourceInstancesWithPageSize(query map[string]string, URL string, pageSize int) ([]*apiv1.ResourceInstance, error)
	GetAPIServiceByName(name string) (*mv1a.APIService, error)
	GetAPIServiceInstanceByName(name string) (*mv1a.APIServiceInstance, error)
	GetAPIRevisionByName(name string) (*mv1a.APIServiceRevision, error)
	CreateCategory(name string) (*catalog.Category, error)
	GetOrCreateCategory(category string) string
	GetEnvironment() (*mv1a.Environment, error)
	GetCentralTeamByName(name string) (*defs.PlatformTeam, error)
	GetTeam(query map[string]string) ([]defs.PlatformTeam, error)
	GetAccessControlList(aclName string) (*mv1a.AccessControlList, error)
	UpdateAccessControlList(acl *mv1a.AccessControlList) (*mv1a.AccessControlList, error)
	CreateAccessControlList(acl *mv1a.AccessControlList) (*mv1a.AccessControlList, error)
	UpdateAPIV1ResourceInstance(url string, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error)
	CreateSubResourceScoped(scopeKindPlural, scopeName, resKindPlural, name, group, version string, subs map[string]interface{}) error
	CreateSubResourceUnscoped(kindPlural, name, group, version string, subs map[string]interface{}) error
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
func (c *ServiceClient) getTeamFromCache(name string) (string, bool) {
	var team *defs.PlatformTeam
	if name == "" {
		team = c.caches.GetDefaultTeam()
		if team == nil {
			return "", false
		}
		return team.ID, true
	}

	team = c.caches.GetTeamByName(name)
	if team == nil {
		return "", false
	}

	return team.ID, true
}

// GetOrCreateCategory - Returns the value on published proxy
func (c *ServiceClient) GetOrCreateCategory(title string) string {
	category := c.caches.GetCategoryWithTitle(title)
	if category == nil {
		if !corecfg.IsCategoryAutocreationEnabled() {
			log.Warnf("Category auto creation is disabled: agent is not allowed to create %s category", title)
			return ""
		}

		// create the category and add it to the cache
		newCategory, err := c.CreateCategory(title)
		if err != nil {
			log.Errorf(errors.Wrap(ErrCategoryCreate, err.Error()).FormatError(title).Error())
			return ""
		}
		category, err = newCategory.AsInstance()
		if err == nil {
			c.caches.AddCategory(category)
		}
	}

	return category.Name
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
func mapToTagsArray(m map[string]interface{}, additionalTags string) []string {
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
	if additionalTags != "" {
		additionalTagsArray := strings.Split(additionalTags, ",")

		for _, tag := range additionalTagsArray {
			strArr = append(strArr, strings.TrimSpace(tag))
		}
	}

	return strArr
}

func readResponseErrors(status int, body []byte) string {
	// Return error string only for error status code
	if status < http.StatusBadRequest {
		return ""
	}

	responseErr := &ResponseError{}
	err := json.Unmarshal(body, &responseErr)
	if err != nil || len(responseErr.Errors) == 0 {
		errStr := getHTTPResponseErrorString(status, body)
		log.Tracef("HTTP response error: %v", string(errStr))
		return errStr
	}

	// Get the first error from the API response errors
	errStr := getAPIResponseErrorString(responseErr.Errors[0])
	log.Tracef("HTTP response error: %s", errStr)
	return errStr
}

func getHTTPResponseErrorString(status int, body []byte) string {
	detail := make(map[string]*json.RawMessage)
	json.Unmarshal(body, &detail)
	errorMsg := ""
	for _, v := range detail {
		buffer, _ := v.MarshalJSON()
		errorMsg = string(buffer)
	}

	errStr := "status - " + strconv.Itoa(status)
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
func (c *ServiceClient) GetEnvironment() (*mv1a.Environment, error) {
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
	env := &mv1a.Environment{}
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
		return nil, errors.Wrap(ErrRequestQuery, responseErr)
	}
}

// GetPlatformUserInfo - request the platform user info
func (c *ServiceClient) getPlatformUserInfo(id string) (*defs.PlatformUserInfo, error) {
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

	var platformUserInfo defs.PlatformUserInfo
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
func (c *ServiceClient) GetCentralTeamByName(name string) (*defs.PlatformTeam, error) {
	// Query for the default, if no teamName is given
	queryParams := map[string]string{}

	if name != "" {
		queryParams = map[string]string{
			"query": fmt.Sprintf("name==\"%s\"", name),
		}
	}

	platformTeams, err := c.GetTeam(queryParams)
	if err != nil {
		return nil, err
	}

	if len(platformTeams) == 0 {
		return nil, ErrTeamNotFound.FormatError(name)
	}

	team := platformTeams[0]
	if name == "" {
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
func (c *ServiceClient) GetTeam(query map[string]string) ([]defs.PlatformTeam, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	// Get the teams using Client registry service instead of from platform.
	// Platform teams API require access and DOSA account will not have the access
	platformURL := fmt.Sprintf("%s/api/v1/platformTeams", c.cfg.GetURL())

	response, reqErr := c.sendServerRequest(platformURL, headers, query)
	if reqErr != nil {
		return nil, reqErr
	}

	var platformTeams []defs.PlatformTeam
	err = json.Unmarshal(response, &platformTeams)
	if err != nil {
		return nil, err
	}

	return platformTeams, nil
}

// GetAccessControlList -
func (c *ServiceClient) GetAccessControlList(name string) (*mv1a.AccessControlList, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/%s", c.cfg.GetEnvironmentACLsURL(), name),
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}

	if response.Code != http.StatusOK {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, errors.Wrap(ErrRequestQuery, responseErr)
	}

	var acl *mv1a.AccessControlList
	err = json.Unmarshal(response.Body, &acl)
	if err != nil {
		return nil, err
	}

	return acl, err
}

//UpdateAccessControlList - removes existing then creates new AccessControlList
func (c *ServiceClient) UpdateAccessControlList(acl *mv1a.AccessControlList) (*mv1a.AccessControlList, error) {
	// first delete the existing access control list
	if _, err := c.deployAccessControl(acl, http.MethodDelete); err != nil {
		return nil, err
	}
	return c.deployAccessControl(acl, http.MethodPost)
}

// CreateAccessControlList -
func (c *ServiceClient) CreateAccessControlList(acl *mv1a.AccessControlList) (*mv1a.AccessControlList, error) {
	return c.deployAccessControl(acl, http.MethodPost)
}

func (c *ServiceClient) deployAccessControl(acl *mv1a.AccessControlList, method string) (*mv1a.AccessControlList, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	url := c.cfg.GetEnvironmentACLsURL()
	if method == http.MethodPut || method == http.MethodDelete {
		url = fmt.Sprintf("%s/%s", url, acl.Name)
	}

	request := coreapi.Request{
		Method:  method,
		URL:     url,
		Headers: headers,
	}

	if method == http.MethodPut || method == http.MethodPost {
		data, err := json.Marshal(*acl)
		if err != nil {
			return nil, err
		}
		request.Body = data
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}

	if response.Code == http.StatusNoContent && method == http.MethodDelete {
		return nil, nil
	}

	if response.Code != http.StatusCreated && response.Code != http.StatusOK {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, errors.Wrap(ErrRequestQuery, responseErr)
	}

	var updatedACL *mv1a.AccessControlList
	if method == http.MethodPut || method == http.MethodPost {
		updatedACL = &mv1a.AccessControlList{}
		err = json.Unmarshal(response.Body, updatedACL)
		if err != nil {
			return nil, err
		}
	}

	return updatedACL, err
}

// ExecuteAPI - execute the api
func (c *ServiceClient) ExecuteAPI(method, url string, query map[string]string, buffer []byte) ([]byte, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:      method,
		URL:         url,
		QueryParams: query,
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
		return nil, errors.Wrap(ErrRequestQuery, responseErr)
	}
}

// linkSubResource creates a sub resource by calling the provided url. url should be the link to the resource.
// subResourceName is the name of the sub resource to add to the resource found at the given url.
// subResourceName will be appended to the end of the url for the PUT request.
// body is the payload of the subresource to create.
func (c *ServiceClient) linkSubResource(url string, body interface{}) error {
	// https://apicentral.axway.com/apis/management/v1alpha1/environments/wc-env/apiservices/eeeee/:extension
	bts, err := json.Marshal(body)
	if err != nil {
		return err
	}

	_, err = c.ExecuteAPI(http.MethodPut, url, nil, bts)
	if err != nil {
		return err
	}

	return nil
}

// CreateSubResourceUnscoped creates a sub resource on th provided unscoped resource.
func (c *ServiceClient) CreateSubResourceUnscoped(
	kindPlural, name, group, version string, subs map[string]interface{},
) error {
	var execErr error
	wg := &sync.WaitGroup{}

	for subName, sub := range subs {
		wg.Add(1)

		base := c.cfg.GetURL()
		url := fmt.Sprintf("%s/apis/%s/%s/%s/%s/%s", base, group, version, kindPlural, name, subName)

		r := map[string]interface{}{
			subName: sub,
		}
		bts, err := json.Marshal(r)
		if err != nil {
			return err
		}

		go func(wg *sync.WaitGroup, sn string) {
			defer wg.Done()
			_, err = c.ExecuteAPI(http.MethodPut, url, nil, bts)
			if err != nil {
				if execErr == nil {
					execErr = err
				}
				log.Errorf("failed to link sub resource %s to resource %s: %v", sn, name, err)
			}
		}(wg, subName)
	}

	wg.Wait()

	return execErr
}

// CreateSubResourceScoped creates a sub resource on th provided scoped resource.
func (c *ServiceClient) CreateSubResourceScoped(
	scopeKindPlural, scopeName, resKindPlural, name, group, version string, subs map[string]interface{},
) error {
	var execErr error
	wg := &sync.WaitGroup{}

	for subName, sub := range subs {
		// if strings.HasPrefix(subName, "x-") {
		wg.Add(1)

		base := c.cfg.GetURL()
		url := fmt.Sprintf("%s/apis/%s/%s/%s/%s/%s/%s/%s", base, group, version, scopeKindPlural, scopeName, resKindPlural, name, subName)

		r := map[string]interface{}{
			subName: sub,
		}
		bts, err := json.Marshal(r)
		if err != nil {
			return err
		}

		go func(sn string) {
			defer wg.Done()
			_, err := c.ExecuteAPI(http.MethodPut, url, nil, bts)
			if err != nil {
				execErr = err
				log.Errorf("failed to link sub resource %s to resource %s: %v", sn, name, err)
			}
		}(subName)
		// }
	}

	wg.Wait()

	return execErr
}
