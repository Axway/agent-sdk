package apic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	v1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	corealpha1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/core/v1alpha1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	utilerrors "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"
	log "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/wsdl"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tidwall/gjson"
)

// createService - creates new APIServerService and necessary resources
// return the itemID from the APIServerService
func (c *ServiceClient) createService(serviceBody ServiceBody) (string, error) {
	externalAPIID := sanitizeAPIName(fmt.Sprintf("%s-%s", serviceBody.RestAPIID, serviceBody.Stage))

	apiServiceInstance, err := c.getAPIServiceByName(serviceBody.RestAPIID)

	if apiServiceInstance != nil {
		log.Infof("API Server Service %s already exists.  Using existing API Server Service", serviceBody.RestAPIID)
	} else {
		// add api
		_, err := c.processService(serviceBody, http.MethodPost, c.cfg.GetServicesURL(), serviceBody.RestAPIID, externalAPIID)
		if err != nil {
			return "", err
		}
	}

	// process revision and instance
	itemID, revisionName, err := c.processRevisionAndInstance(serviceBody, serviceBody.RestAPIID, externalAPIID)
	if err != nil {
		return "", err
	}

	// process consumer instance
	if c.cfg.IsPublishToEnvironmentAndCatalogMode() {
		itemID, err = c.processConsumerInstance(serviceBody, http.MethodPost, c.cfg.GetConsumerInstancesURL(), serviceBody.RestAPIID, externalAPIID, revisionName)
	}
	log.Debugf("Create service returning itemID: [%v]", itemID)
	return itemID, err
}

// updateService - updates APIServerService based on  sanitized name and necessary resources.
// return the itemID from the APIServerService
func (c *ServiceClient) updateService(serviceBody ServiceBody) (string, error) {
	externalAPIID := sanitizeAPIName(fmt.Sprintf("%s-%s", serviceBody.RestAPIID, serviceBody.Stage))

	_, err := c.processService(serviceBody, http.MethodPut, c.cfg.GetServicesURL()+"/"+serviceBody.RestAPIID, serviceBody.RestAPIID, externalAPIID)
	if err != nil {
		return "", err
	}

	itemID, revisionName, err := c.processRevisionAndInstance(serviceBody, serviceBody.RestAPIID, externalAPIID)
	if err != nil {
		return "", err
	}

	// add/update consumer instance
	if c.cfg.IsPublishToEnvironmentAndCatalogMode() {
		consumerInstancesURL := c.cfg.GetConsumerInstancesURL() + "/" + externalAPIID
		itemID, err = c.processConsumerInstance(serviceBody, http.MethodPut, consumerInstancesURL, serviceBody.RestAPIID, externalAPIID, revisionName)
	}

	log.Debugf("Update service returning itemID: [%v]", itemID)
	return itemID, err
}

// processRevisionAndInstance - This is being called from
//	1. createService - when a new api is being published
//	2. updateService - when a major change to an existing api is being published, ie. security profile, version
//		1. add new API Service Revision
//		2. add new API Service Instance
func (c *ServiceClient) processRevisionAndInstance(serviceBody ServiceBody, restAPIID, externalAPIID string) (string, string, error) {
	revisionURL := c.cfg.GetRevisionsURL()
	instanceURL := c.cfg.GetInstancesURL()

	// Get revisions for the service and use the latest one as last reference
	revisionFilter := map[string]string{
		"query": "metadata.references.name==" + restAPIID,
		"sort":  "metadata.audit.createTimestamp,DESC",
	}
	revisions, err := c.getAPIRevisions(revisionFilter)
	if err != nil {
		return "", "", err
	}

	// Derive the name of the revision + instance using the service name and count of existing revisions
	revisionName := externalAPIID + "." + strconv.Itoa(len(revisions)+1)
	httpMethod := http.MethodPost

	var previousRevision *APIServer
	if serviceBody.APIUpdateSeverity == MinorChange {
		// For minor change use the latest revision and update existing
		httpMethod = http.MethodPut
		revisionName = revisions[0].Name
		revisionURL += "/" + revisionName
		instanceURL += "/" + revisionName
	} else if len(revisions) > 0 {
		previousRevision = &revisions[0]
	}

	err = c.processRevision(serviceBody, httpMethod, revisionURL, revisionName, restAPIID, externalAPIID, previousRevision)
	if err != nil {
		return "", "", err
	}

	itemID, err := c.processInstance(serviceBody, httpMethod, instanceURL, revisionName, restAPIID, externalAPIID)
	if err != nil {
		return "", "", err
	}

	return itemID, revisionName, err
}

func isValidAuthPolicy(auth string) bool {
	for _, item := range ValidPolicies {
		if item == auth {
			return true
		}
	}
	return false
}

// getAPIRevisions - Returns the list of API revisions for the specified filter
func (c *ServiceClient) getAPIRevisions(queryParams map[string]string) ([]APIServer, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetRevisionsURL(),
		Headers:     headers,
		QueryParams: queryParams,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		if response.Code != http.StatusNotFound {
			logResponseErrors(response.Body)
		}
		return nil, errors.New(strconv.Itoa(response.Code))
	}
	revisions := make([]APIServer, 0)
	json.Unmarshal(response.Body, &revisions)
	return revisions, nil
}

// getAPIServiceInstanceByName - Returns the API service instance for specified name
func (c *ServiceClient) getAPIServiceInstanceByName(instanceName string) (*APIServer, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.cfg.GetInstancesURL() + "/" + instanceName,
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		if response.Code != http.StatusNotFound {
			logResponseErrors(response.Body)
		}
		return nil, errors.New(strconv.Itoa(response.Code))
	}
	apiInstance := new(APIServer)
	json.Unmarshal(response.Body, apiInstance)
	return apiInstance, nil
}

// getAPIServiceByName - Returns the API service for specified name
func (c *ServiceClient) getAPIServiceByName(name string) (*APIServer, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.cfg.GetServicesURL() + "/" + name,
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		if response.Code != http.StatusNotFound {
			logResponseErrors(response.Body)
		}
		return nil, errors.New(strconv.Itoa(response.Code))
	}
	apiService := new(APIServer)
	json.Unmarshal(response.Body, apiService)
	return apiService, nil
}

// getAPIServerConsumerInstance -
func (c *ServiceClient) getAPIServerConsumerInstance(consumerInstanceName string, queryParams map[string]string) (*APIServer, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	consumerInstanceURL := c.cfg.GetConsumerInstancesURL() + "/" + consumerInstanceName

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         consumerInstanceURL,
		Headers:     headers,
		QueryParams: queryParams,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		if response.Code != http.StatusNotFound {
			logResponseErrors(response.Body)
		}
		return nil, errors.New(strconv.Itoa(response.Code))
	}
	consumerInstance := new(APIServer)
	json.Unmarshal(response.Body, consumerInstance)
	return consumerInstance, nil
}

func (c *ServiceClient) consumerInstanceExists(name string) bool {
	params := map[string]string{
		"fields": "name",
	}
	_, err := c.getAPIServerConsumerInstance(name, params)
	if err != nil {
		if err.Error() != strconv.Itoa(http.StatusNotFound) {
			log.Errorf("Error getting consumerInstance '%v', %v", name, err.Error())
		}
		return false
	}
	return true
}

//processService -
func (c *ServiceClient) processService(serviceBody ServiceBody, httpMethod, servicesURL, restAPIID, externalAPIID string) (string, error) {
	// spec needs to adhere to environment schema
	var spec interface{}
	if serviceBody.Image != "" {
		spec = APIServiceSpec{
			Description: serviceBody.Description,
			Icon: &APIServiceIcon{
				ContentType: serviceBody.ImageContentType,
				Data:        serviceBody.Image,
			},
		}
	} else {
		spec = APIServiceSpec{
			Description: serviceBody.Description,
		}
	}

	serviceBody.NameToPush = serviceBody.APIName
	buffer, err := c.createAPIServerBody(serviceBody, spec, restAPIID, externalAPIID, nil)
	if err != nil {
		return "", err
	}

	return c.apiServiceDeployAPI(httpMethod, servicesURL, buffer)

}

//processRevision -
func (c *ServiceClient) processRevision(serviceBody ServiceBody, httpMethod, revisionsURL, revisionName, restAPIID, externalAPIID string, previousRevision *APIServer) error {
	revisionDefinition := RevisionDefinition{
		Type:  c.getRevisionDefinitionType(serviceBody),
		Value: serviceBody.Swagger,
	}

	spec := APIServiceRevisionSpec{
		APIService: restAPIID,
		Definition: revisionDefinition,
	}

	// If exists, set previous revision as reference for history
	var revAttributes map[string]interface{}
	if previousRevision != nil {
		revAttributes = make(map[string]interface{})
		revAttributes[AttrPreviousAPIServiceRevisionID] = previousRevision.Metadata.ID
	}

	buffer, err := c.createAPIServerBody(serviceBody, spec, revisionName, externalAPIID, revAttributes)
	if err != nil {
		return err
	}

	_, err = c.apiServiceDeployAPI(httpMethod, revisionsURL, buffer)
	if err != nil && httpMethod != http.MethodPut {
		_, err = c.rollbackAPIService(serviceBody, restAPIID, revisionName)
		return err
	}

	return nil
}

//processInstance -
func (c *ServiceClient) processInstance(serviceBody ServiceBody, httpMethod, instancesURL, revisionName, restAPIID, externalAPIID string) (string, error) {
	endPoints, _ := c.getEndpointsBasedOnSwagger(serviceBody.Swagger, c.getRevisionDefinitionType(serviceBody))

	// reset the name here to include the stage
	spec := APIServerInstanceSpec{
		APIServiceRevision: revisionName,
		InstanceEndPoint:   endPoints,
	}

	buffer, err := c.createAPIServerBody(serviceBody, spec, revisionName, externalAPIID, nil)
	if err != nil {
		return "", err
	}

	itemID, err := c.apiServiceDeployAPI(httpMethod, instancesURL, buffer)
	if err != nil && httpMethod != http.MethodPut {
		_, err = c.rollbackAPIService(serviceBody, restAPIID, externalAPIID)
		return "", err
	}

	return itemID, err
}

//processConsumerInstance - deal with either a create or update of a consumerInstance
func (c *ServiceClient) processConsumerInstance(serviceBody ServiceBody, httpMethod, instancesURL, restAPIID, externalAPIID, apiInstanceName string) (string, error) {
	var doc = ""
	apiServerName := externalAPIID
	if serviceBody.Documentation != nil {
		var err error
		doc, err = strconv.Unquote(string(serviceBody.Documentation))
		if err != nil {
			return "", err
		}
	}
	enableSubscription := serviceBody.AuthPolicy != Passthrough

	// if there isn't a registered subscription schema, do not enable subscriptions
	if enableSubscription && c.RegisteredSubscriptionSchema == nil {
		enableSubscription = false
	}

	if enableSubscription {
		log.Debug("Subscriptions will be enabled for consumer instances")
	} else {
		log.Debug("Subscriptions will be disabled for consumer instances, either because the authPolicy is pass-through or there is not a registered subscription schema")
	}

	subscriptionDefinitionName := c.cfg.GetEnvironmentName() + SubscriptionSchemaNameSuffix
	if serviceBody.SubscriptionName != "" {
		subscriptionDefinitionName = serviceBody.SubscriptionName
	}

	autoSubscribe := false
	if c.cfg.GetSubscriptionConfig().GetSubscriptionApprovalMode() == corecfg.AutoApproval {
		autoSubscribe = true
	}

	// Set default state to published
	if serviceBody.State == "" {
		serviceBody.State = PublishedState
	}

	spec := v1alpha1.ConsumerInstanceSpec{
		Name:               serviceBody.NameToPush,
		ApiServiceInstance: apiInstanceName,
		Description:        serviceBody.Description,
		Visibility:         "RESTRICTED",
		Version:            serviceBody.Version,
		State:              string(serviceBody.State),
		Status:             serviceBody.Status,
		Tags:               c.mapToTagsArray(serviceBody.Tags),
		Documentation:      doc,
		OwningTeam:         c.cfg.GetTeamName(),
		Subscription: v1alpha1.ConsumerInstanceSpecSubscription{
			Enabled:                enableSubscription,
			AutoSubscribe:          autoSubscribe,
			SubscriptionDefinition: subscriptionDefinitionName,
		},
	}

	buffer, err := c.createAPIServerBody(serviceBody, spec, apiServerName, externalAPIID, nil)
	if err != nil {
		return "", err
	}

	itemID, err := c.apiServiceDeployAPI(httpMethod, instancesURL, buffer)
	if err != nil && httpMethod != http.MethodPut {
		return c.rollbackAPIService(serviceBody, restAPIID, externalAPIID)
	}

	return itemID, err
}

// rollbackAPIService - if the process to add api/revision/instance fails, delete the api that was created
func (c *ServiceClient) rollbackAPIService(serviceBody ServiceBody, restAPIID, externalAPIID string) (string, error) {
	spec := APIServiceSpec{}
	buffer, err := c.createAPIServerBody(serviceBody, spec, restAPIID, externalAPIID, nil)
	if err != nil {
		return "", err
	}
	return c.apiServiceDeployAPI(http.MethodDelete, c.cfg.DeleteServicesURL()+"/"+externalAPIID, buffer)
}

// deleteConsumerInstance -
func (c *ServiceClient) deleteConsumerInstance(name string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetConsumerInstancesURL()+"/"+name, nil)
	if err != nil && err.Error() != strconv.Itoa(http.StatusNotFound) {
		return err
	}
	return nil
}

// getConsumerInstanceByID
func (c *ServiceClient) getConsumerInstanceByID(instanceID string) (*APIServer, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	log.Debugf("Get consumer instance by id: %s", instanceID)

	params := map[string]string{
		"query": fmt.Sprintf("metadata.id==%s", instanceID),
	}
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetConsumerInstancesURL(),
		Headers:     headers,
		QueryParams: params,
	}

	response, err := c.apiClient.Send(request)

	if err != nil {
		return nil, err
	}
	if !(response.Code == http.StatusOK) {
		logResponseErrors(response.Body)
		return nil, errors.New(strconv.Itoa(response.Code))
	}

	consumerInstances := make([]*APIServer, 0)
	json.Unmarshal(response.Body, &consumerInstances)
	if len(consumerInstances) == 0 {
		return nil, errors.New("Unable to find consumerInstance using instanceID " + instanceID)
	}

	return consumerInstances[0], nil
}

//getRevisionDefinitionType -
func (c *ServiceClient) getRevisionDefinitionType(serviceBody ServiceBody) string {
	var revisionDefinitionType string
	if serviceBody.ResourceType == Wsdl {
		revisionDefinitionType = Wsdl
	} else {
		oasVer := gjson.GetBytes(serviceBody.Swagger, "openapi")
		revisionDefinitionType = Oas2
		if oasVer.Exists() {
			// OAS v3
			revisionDefinitionType = Oas3
		}
	}
	return revisionDefinitionType
}

// createAPIServerBody - create APIServer for server, revision, and instance
func (c *ServiceClient) createAPIServerBody(serviceBody ServiceBody, spec interface{}, apiServerName, externalAPIID string, attributes map[string]interface{}) ([]byte, error) {
	if attributes == nil {
		attributes = make(map[string]interface{})
	}

	attributes[AttrExternalAPIID] = externalAPIID
	attributes[AttrCreatedBy] = serviceBody.CreatedBy

	newtags := c.mapToTagsArray(serviceBody.Tags)

	apiServer := APIServer{
		Name:       apiServerName,
		Title:      serviceBody.NameToPush,
		Attributes: attributes,
		Spec:       spec,
		Tags:       newtags,
	}

	return json.Marshal(apiServer)
}

func (c *ServiceClient) getEndpointsBasedOnSwagger(swagger []byte, revisionDefinitionType string) ([]EndPoint, error) {
	switch revisionDefinitionType {
	case Wsdl:
		return c.getWsdlEndpoints(swagger)
	case Oas2:
		return c.getOas2Endpoints(swagger)
	case Oas3:
		return c.getOas3Endpoints(swagger)
	}

	return nil, fmt.Errorf("Unable to get endpoints from swagger; invalid definition type: %v", revisionDefinitionType)
}

func (c *ServiceClient) getWsdlEndpoints(swagger []byte) ([]EndPoint, error) {
	endPoints := []EndPoint{}
	def, err := wsdl.Unmarshal(swagger)
	if err != nil {
		log.Errorf("Error unmarshalling WSDL to get endpoints: %v", err.Error())
		return nil, err
	}

	ports := def.Service.Ports
	for _, val := range ports {
		loc := val.Address.Location
		fixed, err := url.Parse(loc)
		if err != nil {
			log.Errorf("Error parsing service location in WSDL to get endpoints: %v", err.Error())
			return nil, err
		}
		protocol := fixed.Scheme
		host := fixed.Hostname()
		portStr := fixed.Port()
		if portStr == "" {
			p, err := net.LookupPort("tcp", protocol)
			if err != nil {
				log.Errorf("Error finding port for endpoint: %v", err.Error())
				return nil, err
			}
			portStr = strconv.Itoa(p)
		}
		port, _ := strconv.Atoi(portStr)

		basePath := BasePath{
			Path: fixed.Path,
		}

		endPoint := EndPoint{
			Host:     host,
			Port:     port,
			Protocol: protocol,
			Routing:  basePath,
		}
		if !contains(endPoints, endPoint) {
			endPoints = append(endPoints, endPoint)
		}
	}

	return endPoints, nil
}

func contains(endpts []EndPoint, endpt EndPoint) bool {
	for _, pt := range endpts {
		if pt == endpt {
			return true
		}
	}
	return false
}
func (c *ServiceClient) getOas2Endpoints(swagger []byte) ([]EndPoint, error) {
	endPoints := []EndPoint{}
	swaggerHost := strings.Split(gjson.Get(string(swagger), "host").String(), ":")
	host := swaggerHost[0]
	port := 443
	if len(swaggerHost) > 1 {
		swaggerPort, err := strconv.Atoi(swaggerHost[1])
		if err == nil {
			port = swaggerPort
		}
	}

	schemes := make([]string, 0)
	protocols := gjson.Get(string(swagger), "schemes")
	err := json.Unmarshal([]byte(protocols.Raw), &schemes)
	if err != nil {
		log.Errorf("Error getting schemas from Swagger 2.0 definition: %s", err.Error())
		return nil, err
	}

	basePath := BasePath{
		Path: gjson.Get(string(swagger), "basePath").String(),
	}

	for _, protocol := range schemes {
		endPoint := EndPoint{
			Host:     host,
			Port:     port,
			Protocol: protocol,
			Routing:  basePath,
		}
		endPoints = append(endPoints, endPoint)
	}

	return endPoints, nil
}

func (c *ServiceClient) getOas3Endpoints(swagger []byte) ([]EndPoint, error) {
	endPoints := []EndPoint{}
	openAPI, _ := openapi3.NewSwaggerLoader().LoadSwaggerFromData(swagger)

	for _, server := range openAPI.Servers {
		// Add the URL string to the array
		allURLs := []string{
			server.URL,
		}

		defaultURL := ""
		var err error
		if server.Variables != nil {
			defaultURL, allURLs, err = c.handleURLSubstitutions(server, allURLs)
			if err != nil {
				return nil, err
			}
		}

		parsedEndPoints, err := c.parseURLsIntoEndpoints(defaultURL, allURLs)
		if err != nil {
			return nil, err
		}
		endPoints = append(endPoints, parsedEndPoints...)
	}

	return endPoints, nil
}

func (c *ServiceClient) handleURLSubstitutions(server *openapi3.Server, allURLs []string) (string, []string, error) {
	defaultURL := server.URL
	// Handle substitutions
	for serverKey, serverVar := range server.Variables {
		newURLs := []string{}
		if serverVar.Default == nil {
			err := fmt.Errorf("Server variable in OAS3 %s does not have a default value, spec not valid", serverKey)
			log.Errorf(err.Error())
			return "", nil, err
		}
		defaultURL = strings.ReplaceAll(defaultURL, fmt.Sprintf("{%s}", serverKey), serverVar.Default.(string))
		if len(serverVar.Enum) == 0 {
			for _, template := range allURLs {
				newURLs = append(newURLs, strings.ReplaceAll(template, fmt.Sprintf("{%s}", serverKey), serverVar.Default.(string)))
			}
		} else {
			for _, enumVal := range serverVar.Enum {
				for _, template := range allURLs {
					newURLs = append(newURLs, strings.ReplaceAll(template, fmt.Sprintf("{%s}", serverKey), enumVal.(string)))
				}
			}
		}
		allURLs = newURLs
	}

	return defaultURL, allURLs, nil
}

func (c *ServiceClient) parseURLsIntoEndpoints(defaultURL string, allURLs []string) ([]EndPoint, error) {
	endPoints := []EndPoint{}
	for _, urlStr := range allURLs {
		urlObj, err := url.Parse(urlStr)
		if err != nil {
			err := fmt.Errorf("Could not parse url: %s", urlStr)
			log.Errorf(err.Error())
			return nil, err
		}
		// If a port is not given, use lookup the default
		var port int
		if urlObj.Port() == "" {
			port, _ = net.LookupPort("tcp", urlObj.Scheme)
		} else {
			port, _ = strconv.Atoi(urlObj.Port())
		}

		basePath := BasePath{
			Path: urlObj.Path,
		}

		endPoint := EndPoint{
			Host:     urlObj.Hostname(),
			Port:     port,
			Protocol: urlObj.Scheme,
			Routing:  basePath,
		}

		// If the URL is the default URL put it at the front of the array
		if urlStr == defaultURL {
			newEndPoints := []EndPoint{endPoint}
			for _, oldEndpoint := range endPoints {
				newEndPoints = append(newEndPoints, oldEndpoint)
			}
			endPoints = newEndPoints
		} else {
			endPoints = append(endPoints, endPoint)
		}
	}

	return endPoints, nil
}

// Sanitize name to be path friendly and follow this regex: ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*
func sanitizeAPIName(name string) string {
	// convert all letters to lower first
	newName := strings.ToLower(name)

	// parse name out. All valid parts must be '-', '.', a-z, or 0-9
	re := regexp.MustCompile(`[-\.a-z0-9]*`)
	matches := re.FindAllString(newName, -1)

	// join all of the parts, separated with '-'. This in effect is substituting all illegal chars with a '-'
	newName = strings.Join(matches, "-")

	// The regex rule says that the name must not begin or end with a '-' or '.', so trim them off
	newName = strings.TrimLeft(strings.TrimRight(newName, "-."), "-.")

	// The regex rule also says that the name must not have a sequence of ".-", "-.", or "..", so replace them
	r1 := strings.ReplaceAll(newName, "-.", "--")
	r2 := strings.ReplaceAll(r1, ".-", "--")
	r3 := strings.ReplaceAll(r2, "..", "--")

	return r3
}

// apiServiceDeployAPI -
func (c *ServiceClient) apiServiceDeployAPI(method, url string, buffer []byte) (string, error) {
	headers, err := c.createHeader()
	if err != nil {
		return "", err
	}

	request := coreapi.Request{
		Method:      method,
		URL:         url,
		QueryParams: nil,
		Headers:     headers,
		Body:        buffer,
	}
	response, err := c.apiClient.Send(request)
	if err != nil {
		return "", err
	}
	//  Check to see if rollback was processed
	if method == http.MethodDelete && response.Code == http.StatusNoContent {
		log.Debug("API service has been removed.")
		logResponseErrors(response.Body)
		return "", errors.New(strconv.Itoa(response.Code))
	}

	if !(response.Code == http.StatusOK || response.Code == http.StatusCreated) {
		logResponseErrors(response.Body)
		return "", errors.New(strconv.Itoa(response.Code))
	}

	itemID := ""
	metadata := gjson.Get(string(response.Body), "metadata").String()
	if metadata != "" {
		itemID = gjson.Get(string(metadata), "id").String()
	}

	return itemID, nil
}

// RegisterSubscriptionWebhook - Adds a new Subscription webhook. There is a single webhook
// per environment
func (c *ServiceClient) RegisterSubscriptionWebhook() error {
	// if the default is already set up, do nothing
	webhookCfg := c.cfg.GetSubscriptionConfig().GetSubscriptionApprovalWebhookConfig()
	if webhookCfg == nil || !webhookCfg.IsConfigured() {
		return nil
	}

	// create the secret
	err := c.createSecret()
	if err != nil {
		return utilerrors.Wrap(ErrCreateSecret, err.Error())

	}

	err = c.createWebhook()
	if err != nil {
		return utilerrors.Wrap(ErrCreateWebhook, err.Error())
	}

	return nil
}

// create the on-and-only secret for the environment
func (c *ServiceClient) createSecret() error {
	s := c.DefaultSubscriptionApprovalWebhook.GetSecret()
	spec := corealpha1.SecretSpec{
		Data: map[string]string{DefaultSubscriptionWebhookAuthKey: base64.StdEncoding.EncodeToString([]byte(s))},
	}

	secret := corealpha1.Secret{
		ResourceMeta: v1.ResourceMeta{Name: DefaultSubscriptionWebhookName},
		Spec:         spec,
	}

	buffer, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	headers, err := c.createHeader()
	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     c.cfg.GetAPIServerSecretsURL(),
		Headers: headers,
		Body:    buffer,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return err
	}
	if !(response.Code == http.StatusCreated || response.Code == http.StatusConflict) {
		logResponseErrors(response.Body)
		return errors.New(strconv.Itoa(response.Code))
	}
	if response.Code == http.StatusConflict {
		request = coreapi.Request{
			Method:  coreapi.PUT,
			URL:     c.cfg.GetAPIServerSecretsURL() + "/" + DefaultSubscriptionWebhookName,
			Headers: headers,
			Body:    buffer,
		}

		response, err := c.apiClient.Send(request)
		if err != nil {
			return err
		}
		if !(response.Code == http.StatusOK) {
			logResponseErrors(response.Body)
			return errors.New(strconv.Itoa(response.Code))
		}
	}

	return nil
}

// create the on-and-only subscription approval webhook for the environment
func (c *ServiceClient) createWebhook() error {
	webhookCfg := c.cfg.GetSubscriptionConfig().GetSubscriptionApprovalWebhookConfig()
	specSecret := corealpha1.WebhookSpecAuthSecret{
		Name: DefaultSubscriptionWebhookName,
		Key:  DefaultSubscriptionWebhookAuthKey,
	}
	authSpec := corealpha1.WebhookSpecAuth{
		Secret: specSecret,
	}
	webSpec := corealpha1.WebhookSpec{
		Auth:    authSpec,
		Enabled: true,
		Url:     webhookCfg.GetURL(),
		Headers: webhookCfg.GetWebhookHeaders(),
	}

	webhook := corealpha1.Webhook{
		ResourceMeta: v1.ResourceMeta{Name: DefaultSubscriptionWebhookName},
		Spec:         webSpec,
	}

	buffer, err := json.Marshal(webhook)
	if err != nil {
		return err
	}

	headers, err := c.createHeader()
	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     c.cfg.GetAPIServerWebhooksURL(),
		Headers: headers,
		Body:    buffer,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return err
	}
	if !(response.Code == http.StatusCreated || response.Code == http.StatusConflict) {
		logResponseErrors(response.Body)
		return errors.New(strconv.Itoa(response.Code))
	}
	if response.Code == http.StatusConflict {
		request = coreapi.Request{
			Method:  coreapi.PUT,
			URL:     c.cfg.GetAPIServerWebhooksURL() + "/" + DefaultSubscriptionWebhookName,
			Headers: headers,
			Body:    buffer,
		}

		response, err := c.apiClient.Send(request)
		if err != nil {
			return err
		}
		if !(response.Code == http.StatusOK) {
			logResponseErrors(response.Body)
			return errors.New(strconv.Itoa(response.Code))
		}
	}

	return nil
}
