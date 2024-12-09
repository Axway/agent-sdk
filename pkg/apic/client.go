package apic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"

	cache2 "github.com/Axway/agent-sdk/pkg/agent/cache"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
	Basic       = "http-basic"
)

// other consts
const (
	TeamMapKey = "TeamMap"
)

// constants for patch request
const (
	PatchOpAdd             = "add"
	PatchOpReplace         = "replace"
	PatchOpDelete          = "delete"
	PatchOpBuildObjectTree = "x-build-object-tree"
	PatchOperation         = "op"
	PatchPath              = "path"
	PatchValue             = "value"
	ContentTypeJsonPatch   = "application/json-patch+json"
	ContentTypeJson        = "application/json"
)

// constants for patch request
const (
	BearerTokenPrefix = "Bearer "
	HdrContentType    = "Content-Type"
	HdrAuthorization  = "Authorization"
	HdrAxwayTenantID  = "X-Axway-Tenant-Id"
)

// ValidPolicies - list of valid auth policies supported by Central.  Add to this list as more policies are supported.
var ValidPolicies = []string{Apikey, Passthrough, Oauth, Basic}

// Client - interface
type Client interface {
	SetTokenGetter(tokenRequester auth.PlatformTokenGetter)
	SetConfig(cfg corecfg.CentralConfig)
	PublishService(serviceBody *ServiceBody) (*management.APIService, error)
	DeleteAPIServiceInstance(name string) error
	DeleteServiceByName(name string) error
	GetUserEmailAddress(ID string) (string, error)
	GetUserName(ID string) (string, error)
	ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
	Healthcheck(name string) *hc.Status
	GetAPIRevisions(query map[string]string, stage string) ([]*management.APIServiceRevision, error)
	GetAPIServiceRevisions(query map[string]string, URL, stage string) ([]*management.APIServiceRevision, error)
	GetAPIServiceInstances(query map[string]string, URL string) ([]*management.APIServiceInstance, error)
	GetAPIV1ResourceInstances(query map[string]string, URL string) ([]*apiv1.ResourceInstance, error)
	GetAPIV1ResourceInstancesWithPageSize(query map[string]string, URL string, pageSize int) ([]*apiv1.ResourceInstance, error)
	GetAPIServiceByName(name string) (*management.APIService, error)
	GetAPIServiceInstanceByName(name string) (*management.APIServiceInstance, error)
	GetAPIRevisionByName(name string) (*management.APIServiceRevision, error)
	GetEnvironment() (*management.Environment, error)
	GetCentralTeamByName(name string) (*defs.PlatformTeam, error)
	GetTeam(query map[string]string) ([]defs.PlatformTeam, error)
	GetAccessControlList(aclName string) (*management.AccessControlList, error)
	UpdateAccessControlList(acl *management.AccessControlList) (*management.AccessControlList, error)
	CreateAccessControlList(acl *management.AccessControlList) (*management.AccessControlList, error)

	CreateSubResource(rm apiv1.ResourceMeta, subs map[string]interface{}) error
	GetResource(url string) (*apiv1.ResourceInstance, error)
	UpdateResourceFinalizer(ri *apiv1.ResourceInstance, finalizer, description string, addAction bool) (*apiv1.ResourceInstance, error)

	UpdateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	CreateOrUpdateResource(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	CreateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	PatchSubResource(ri apiv1.Interface, subResourceName string, patches []map[string]interface{}) (*apiv1.ResourceInstance, error)
	DeleteResourceInstance(ri apiv1.Interface) error
	GetResources(ri apiv1.Interface) ([]apiv1.Interface, error)

	CreateResource(url string, bts []byte) (*apiv1.ResourceInstance, error)
	UpdateResource(url string, bts []byte) (*apiv1.ResourceInstance, error)
}

// New creates a new Client
func New(cfg corecfg.CentralConfig, tokenRequester auth.PlatformTokenGetter, caches cache2.Manager) Client {
	serviceClient := &ServiceClient{
		caches:        caches,
		pageSizes:     map[string]int{},
		pageSizeMutex: &sync.Mutex{},
	}
	serviceClient.logger = log.NewFieldLogger().
		WithComponent("serviceClient").
		WithPackage("sdk.apic")

	serviceClient.SetTokenGetter(tokenRequester)
	serviceClient.subscriptionSchemaCache = cache.New()
	serviceClient.initClient(cfg)

	return serviceClient
}

func (c *ServiceClient) createAPIServerURL(link string) string {
	return fmt.Sprintf("%s/apis%s", c.cfg.GetURL(), link)
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

// initClient - config change handler
func (c *ServiceClient) initClient(cfg corecfg.CentralConfig) {
	c.cfg = cfg
	c.apiClient = coreapi.NewClient(cfg.GetTLSConfig(), cfg.GetProxyURL(),
		coreapi.WithTimeout(cfg.GetClientTimeout()), coreapi.WithSingleURL())

	err := c.setTeamCache()
	if err != nil {
		c.logger.Error(err)
	}

}

// SetTokenGetter - sets the token getter
func (c *ServiceClient) SetTokenGetter(tokenRequester auth.PlatformTokenGetter) {
	c.tokenRequester = tokenRequester
}

// SetConfig - sets the config and apiClient
func (c *ServiceClient) SetConfig(cfg corecfg.CentralConfig) {
	c.cfg = cfg
	c.apiClient = coreapi.NewClient(cfg.GetTLSConfig(), cfg.GetProxyURL(),
		coreapi.WithTimeout(cfg.GetClientTimeout()), coreapi.WithSingleURL())
}

// mapToTagsArray -
func mapToTagsArray(m map[string]interface{}, additionalTags string) []string {
	strArr := []string{}

	for key, val := range m {
		value := key
		if v, ok := val.(*string); ok && *v != "" {
			value += "_" + *v
		} else if v, ok := val.(string); ok && v != "" {
			value += "_" + v
		}
		if len(value) > 80 {
			value = value[:77] + "..."
		}
		strArr = append(strArr, value)
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
	headers[HdrAxwayTenantID] = c.cfg.GetTenantID()
	headers[HdrAuthorization] = BearerTokenPrefix + token
	headers[HdrContentType] = ContentTypeJson
	return headers, nil
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
func (c *ServiceClient) GetEnvironment() (*management.Environment, error) {
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
	env := &management.Environment{}
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
	c.logger.Tracef("Platform URL being used to get user information %s", platformURL)

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
	c.logger.Tracef("Platform user email %s", email)

	return email, nil
}

// GetUserName - request the user name
func (c *ServiceClient) GetUserName(id string) (string, error) {
	platformUserInfo, err := c.getPlatformUserInfo(id)
	if err != nil {
		return "", err
	}

	userName := fmt.Sprintf("%s %s", platformUserInfo.Result.Firstname, platformUserInfo.Result.Lastname)

	c.logger.Tracef("Platform user %s", userName)

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
func (c *ServiceClient) GetAccessControlList(name string) (*management.AccessControlList, error) {
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

	var acl *management.AccessControlList
	err = json.Unmarshal(response.Body, &acl)
	if err != nil {
		return nil, err
	}

	return acl, err
}

// UpdateAccessControlList - removes existing then creates new AccessControlList
func (c *ServiceClient) UpdateAccessControlList(acl *management.AccessControlList) (*management.AccessControlList, error) {
	return c.deployAccessControl(acl, http.MethodPut)
}

// CreateAccessControlList -
func (c *ServiceClient) CreateAccessControlList(acl *management.AccessControlList) (*management.AccessControlList, error) {
	return c.deployAccessControl(acl, http.MethodPost)
}

func (c *ServiceClient) deployAccessControl(acl *management.AccessControlList, method string) (*management.AccessControlList, error) {
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

	if method == http.MethodDelete && (response.Code == http.StatusNotFound || response.Code == http.StatusNoContent) {
		return nil, nil
	}

	if response.Code == http.StatusConflict {
		curACL, _ := c.GetResource(acl.GetSelfLink())
		c.caches.SetAccessControlList(curACL)
	}

	if response.Code != http.StatusCreated && response.Code != http.StatusOK {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, errors.Wrap(ErrRequestQuery, responseErr)
	}

	var updatedACL *management.AccessControlList
	if method == http.MethodPut || method == http.MethodPost {
		updatedACL = &management.AccessControlList{}
		err = json.Unmarshal(response.Body, updatedACL)
		if err != nil {
			return nil, err
		}
	}

	return updatedACL, err
}

// executeAPI - execute the api
func (c *ServiceClient) executeAPI(method, url string, query map[string]string, buffer []byte, overrideHeaders map[string]string) (*coreapi.Response, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	for key, value := range overrideHeaders {
		headers[key] = value
	}

	request := coreapi.Request{
		Method:      method,
		URL:         url,
		QueryParams: query,
		Headers:     headers,
		Body:        buffer,
	}

	return c.apiClient.Send(request)
}

// ExecuteAPI - execute the api
func (c *ServiceClient) ExecuteAPI(method, url string, query map[string]string, buffer []byte) ([]byte, error) {
	return c.ExecuteAPIWithHeader(method, url, query, buffer, nil)
}

// ExecuteAPI - execute the api
func (c *ServiceClient) ExecuteAPIWithHeader(method, url string, query map[string]string, buffer []byte, headers map[string]string) ([]byte, error) {
	response, err := c.executeAPI(method, url, query, buffer, headers)
	if err != nil {
		return nil, errors.Wrap(ErrNetwork, err.Error())
	}

	switch {
	case response.Code == http.StatusNoContent && method == http.MethodDelete:
		return nil, nil
	case response.Code == http.StatusOK:
		fallthrough
	case response.Code == http.StatusCreated:
		return response.Body, nil
	case response.Code == http.StatusUnauthorized:
		return nil, ErrAuthentication
	default:
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, errors.Wrap(ErrRequestQuery, responseErr)
	}
}

// CreateSubResource creates a sub resource on the provided resource.
func (c *ServiceClient) CreateSubResource(rm apiv1.ResourceMeta, subs map[string]interface{}) error {
	_, err := c.createSubResource(rm, subs)
	return err
}

func (c *ServiceClient) createSubResource(rm apiv1.ResourceMeta, subs map[string]interface{}) (*apiv1.ResourceInstance, error) {
	var execErr error
	var instanceBytes []byte
	wg := &sync.WaitGroup{}
	bytesMutex := &sync.Mutex{}

	subsToUpdate := map[string]string{}
	for subName, sub := range subs {
		if existingHash, ok := rm.GetSubResourceHash(subName); ok {
			hash, err := util.ComputeHash(sub)
			if err == nil && float64(hash) == existingHash {
				c.logger.WithField("resourceName", rm.Name).WithField("subResourceName", subName).Debug("hash found, skipping createSubResource")
				continue
			}
		}
		rm.SetSubResource(subName, sub)
		subsToUpdate[subName] = ""
		subsToUpdate[definitions.XAgentDetails] = ""
	}

	rm.PrepareHashesForSending()
	for subName, _ := range subsToUpdate {
		wg.Add(1)
		url := c.createAPIServerURL(fmt.Sprintf("%s/%s", rm.GetSelfLink(), subName))

		r := map[string]interface{}{
			subName: rm.GetSubResource(subName),
		}
		bts, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}

		go func(sn string) {
			defer wg.Done()
			var err error
			bytesMutex.Lock()
			instanceBytes, err = c.ExecuteAPI(http.MethodPut, url, nil, bts)
			if err != nil {
				execErr = err
				c.logger.Errorf("failed to link sub resource %s to resource %s: %v", sn, rm.Name, err)
			}
			bytesMutex.Unlock()
		}(subName)
	}
	wg.Wait()
	rm.SetIncomingHashes()

	if execErr != nil {
		return nil, execErr
	} else if len(instanceBytes) == 0 {
		c.logger.WithField("resourceName", rm.Name).Debug("no subResource updates were executed")
		return nil, nil
	}

	ri := &apiv1.ResourceInstance{}
	err := json.Unmarshal(instanceBytes, ri)
	if err != nil {
		return nil, err
	}

	return ri, nil
}

// GetResource gets a single resource
func (c *ServiceClient) GetResource(url string) (*apiv1.ResourceInstance, error) {
	response, err := c.ExecuteAPI(http.MethodGet, c.createAPIServerURL(url), nil, nil)
	if err != nil {
		return nil, err
	}
	ri := &apiv1.ResourceInstance{}
	err = json.Unmarshal(response, ri)
	return ri, err
}

// GetResource gets a single resource
func (c *ServiceClient) GetResources(iface apiv1.Interface) ([]apiv1.Interface, error) {
	inst, err := iface.AsInstance()
	if err != nil {
		return nil, err
	}

	response, err := c.ExecuteAPI(http.MethodGet, c.createAPIServerURL(inst.GetKindLink()), nil, nil)
	if err != nil {
		return nil, err
	}

	instances := []*apiv1.ResourceInstance{}
	err = json.Unmarshal(response, &instances)
	if err != nil {
		return nil, err
	}

	ifaces := []apiv1.Interface{}
	for i := range instances {
		ifaces = append(ifaces, instances[i])
	}
	return ifaces, nil
}

// UpdateResourceFinalizer - Add or remove a finalizer from a resource
func (c *ServiceClient) UpdateResourceFinalizer(res *apiv1.ResourceInstance, finalizer, description string, addAction bool) (*apiv1.ResourceInstance, error) {
	if addAction {
		res.Finalizers = append(res.Finalizers, apiv1.Finalizer{Name: finalizer, Description: description})
	} else {
		cleanedFinalizer := make([]apiv1.Finalizer, 0)
		for _, f := range res.Finalizers {
			if f.Name != finalizer {
				cleanedFinalizer = append(cleanedFinalizer, f)
			}
		}
		res.Finalizers = cleanedFinalizer
	}

	return c.UpdateResourceInstance(res)
}

// UpdateResource updates a resource
func (c *ServiceClient) UpdateResource(url string, bts []byte) (*apiv1.ResourceInstance, error) {
	log.DeprecationWarningReplace("UpdateResource", "UpdateResourceInstance")

	response, err := c.ExecuteAPI(http.MethodPut, c.createAPIServerURL(url), nil, bts)
	if err != nil {
		return nil, err
	}
	ri := &apiv1.ResourceInstance{}
	err = json.Unmarshal(response, ri)
	return ri, err
}

// CreateResource deletes a resource
func (c *ServiceClient) CreateResource(url string, bts []byte) (*apiv1.ResourceInstance, error) {
	log.DeprecationWarningReplace("CreateResource", "CreateResourceInstance")

	response, err := c.ExecuteAPI(http.MethodPost, c.createAPIServerURL(url), nil, bts)
	if err != nil {
		return nil, err
	}
	ri := &apiv1.ResourceInstance{}
	err = json.Unmarshal(response, ri)
	return ri, err
}

func (c *ServiceClient) getCachedResource(data *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	switch data.Kind {
	case management.AccessRequestDefinitionGVK().Kind:
		return c.caches.GetAccessRequestDefinitionByName(data.Name)
	case management.CredentialRequestDefinitionGVK().Kind:
		return c.caches.GetCredentialRequestDefinitionByName(data.Name)
	case management.APIServiceInstanceGVK().Kind:
		return c.caches.GetAPIServiceInstanceByName(data.Name)
	}
	return nil, nil
}

func (c *ServiceClient) addResourceToCache(data *apiv1.ResourceInstance) {
	switch data.Kind {
	case management.AccessRequestDefinitionGVK().Kind:
		c.caches.AddAccessRequestDefinition(data)
	case management.CredentialRequestDefinitionGVK().Kind:
		c.caches.AddCredentialRequestDefinition(data)
	case management.APIServiceInstanceGVK().Kind:
		c.caches.AddAPIServiceInstance(data)
	}
}

// updateORCreateResourceInstance
func (c *ServiceClient) updateSpecORCreateResourceInstance(data *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	// default to post
	url := c.createAPIServerURL(data.GetKindLink())
	method := coreapi.POST

	// check if the KIND and ID combo have an item in the cache
	existingRI, err := c.getCachedResource(data)
	updateRI := true
	updateAgentDetails := true

	if err == nil && existingRI != nil && existingRI.Metadata.Scope.Name == data.Metadata.Scope.Name {
		url = c.createAPIServerURL(data.GetSelfLink())
		method = coreapi.PUT

		// check if either hash or title has changed and mark for update
		oldHash, _ := util.GetAgentDetailsValue(existingRI, defs.AttrSpecHash)
		newHash, _ := util.GetAgentDetailsValue(data, defs.AttrSpecHash)
		if oldHash == newHash && existingRI.Title == data.Title {
			log.Debug("no updates to the hash or to the title")
			updateRI = false
		}

		// check if x-agent-details have changed and mark for update
		oldAgentDetails := util.GetAgentDetails(existingRI)
		newAgentDetails := util.GetAgentDetails(data)
		if util.MapsEqual(oldAgentDetails, newAgentDetails) {
			log.Debug("no updates to the x-agent-details")
			updateAgentDetails = false
		}

		// if no changes altogether, return without update
		if !updateRI && !updateAgentDetails {
			log.Trace("no updates made to the resource instance or to the x-agent-details.")
			return existingRI, nil
		}

		// Update the spec and agent details subresource, if they exist in incoming data
		existingRI.Spec = data.Spec
		existingRI.SubResources = data.SubResources
		existingRI.Title = data.Title
		existingRI.Metadata.ResourceVersion = ""

		// set the data and subresources to be pushed
		data = existingRI
	}

	newRI := &apiv1.ResourceInstance{}
	if updateRI {
		reqBytes, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		response, err := c.ExecuteAPI(method, url, nil, reqBytes)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(response, newRI)
		if err != nil {
			return nil, err
		}
	} else {
		newRI = existingRI
	}

	if data := util.GetAgentDetails(data); data != nil && updateAgentDetails {
		var receivedRI *apiv1.ResourceInstance
		// only send in the agent details here, that is all the agent needs to update for anything here
		receivedRI, err = c.createSubResource(newRI.ResourceMeta, map[string]interface{}{defs.XAgentDetails: data})
		if err != nil {
			return nil, err
		} else if receivedRI != nil {
			newRI = receivedRI
		}
	}

	if existingRI == nil {
		c.addResourceToCache(newRI)
	}
	return newRI, err
}

// CreateOrUpdateResource deletes a resource
func (c *ServiceClient) CreateOrUpdateResource(data apiv1.Interface) (*apiv1.ResourceInstance, error) {
	data.SetScopeName(c.cfg.GetEnvironmentName())
	ri, err := data.AsInstance()
	if err != nil {
		return nil, err
	}

	ri, err = c.updateSpecORCreateResourceInstance(ri)
	return ri, err
}

// UpdateResourceInstance - updates a ResourceInstance
func (c *ServiceClient) UpdateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
	inst, err := ri.AsInstance()
	if err != nil {
		return nil, err
	}
	if inst.GetSelfLink() == "" {
		return nil, fmt.Errorf("could not remove resource instance, could not get self link")
	}
	inst.Metadata.ResourceVersion = ""
	bts, err := json.Marshal(ri)
	if err != nil {
		return nil, err
	}
	bts, err = c.ExecuteAPI(coreapi.PUT, c.createAPIServerURL(inst.GetSelfLink()), nil, bts)
	if err != nil {
		return nil, err
	}
	r := &apiv1.ResourceInstance{}
	err = json.Unmarshal(bts, r)
	return r, err
}

// DeleteResourceInstance - deletes a ResourceInstance
func (c *ServiceClient) DeleteResourceInstance(ri apiv1.Interface) error {
	inst, err := ri.AsInstance()
	if err != nil {
		return err
	}
	if inst.GetSelfLink() == "" {
		return fmt.Errorf("could not remove resource instance, could not get self link")
	}
	bts, err := json.Marshal(ri)
	if err != nil {
		return err
	}
	_, err = c.ExecuteAPI(coreapi.DELETE, c.createAPIServerURL(inst.GetSelfLink()), nil, bts)
	return err
}

// CreateResourceInstance - creates a ResourceInstance
func (c *ServiceClient) CreateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
	inst, err := ri.AsInstance()
	if err != nil {
		return nil, err
	}
	if inst.GetKindLink() == "" {
		return nil, fmt.Errorf("could not create resource instance, could not get self link")
	}
	inst.Metadata.ResourceVersion = ""
	bts, err := json.Marshal(ri)
	if err != nil {
		return nil, err
	}
	bts, err = c.ExecuteAPI(coreapi.POST, c.createAPIServerURL(inst.GetKindLink()), nil, bts)
	if err != nil {
		return nil, err
	}
	r := &apiv1.ResourceInstance{}
	err = json.Unmarshal(bts, r)
	return r, err
}

// PatchSubResource - applies the patches to the sub-resource
func (c *ServiceClient) PatchSubResource(ri apiv1.Interface, subResourceName string, patches []map[string]interface{}) (*apiv1.ResourceInstance, error) {
	inst, err := ri.AsInstance()
	if err != nil {
		return nil, err
	}

	if inst.GetSelfLink() == "" {
		return nil, fmt.Errorf("could not apply patch to resource instance, unable to get self link")
	}

	// no patches to be applied
	if len(patches) == 0 {
		return inst, nil
	}

	p := make([]map[string]interface{}, 0)
	// add operation to build object tree to allow api-server
	// to expand the sub-resources while applying the patch
	p = append(p, map[string]interface{}{
		PatchOperation: PatchOpBuildObjectTree,
		PatchPath:      fmt.Sprintf("/%s", subResourceName),
	})

	p = append(p, patches...)

	bts, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	url := c.createAPIServerURL(fmt.Sprintf("%s/%s", inst.GetSelfLink(), subResourceName))
	bts, err = c.ExecuteAPIWithHeader(coreapi.PATCH, url, nil, bts, map[string]string{HdrContentType: ContentTypeJsonPatch})
	if err != nil {
		return nil, err
	}

	r := &apiv1.ResourceInstance{}
	err = json.Unmarshal(bts, r)
	return r, err
}
