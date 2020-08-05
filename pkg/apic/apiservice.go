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

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	log "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/wsdl"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tidwall/gjson"
)

// processAPIService - This function will add or update the api service
// If the api doesn't exist, it will add the service, revision, and instance.
// If the api does exist, it will update the revision and instance
func (c *ServiceClient) processAPIService(serviceBody ServiceBody) (string, error) {
	if !isValidAuthPolicy(serviceBody.AuthPolicy) {
		return "", fmt.Errorf("Unsupported security policy '%v'. ", serviceBody.AuthPolicy)
	}

	itemID := ""
	var err error
	httpMethod := http.MethodPut
	sanitizedName := sanitizeAPIName(serviceBody.APIName + serviceBody.Stage)
	servicesURL := c.cfg.GetAPIServerServicesURL() + "/" + sanitizedName
	revisionsURL := c.cfg.GetAPIServerServicesRevisionsURL() + "/" + sanitizedName
	serviceInstancesURL := c.cfg.GetAPIServerServicesInstancesURL() + "/" + sanitizedName
	consumerInstancesURL := c.cfg.GetAPIServerConsumerInstancesURL() + "/" + sanitizedName

	// Verify if the api already exists
	if c.isNewAPI(serviceBody) {
		// add api
		httpMethod = http.MethodPost
		servicesURL := c.cfg.GetAPIServerServicesURL()
		revisionsURL = c.cfg.GetAPIServerServicesRevisionsURL()
		serviceInstancesURL = c.cfg.GetAPIServerServicesInstancesURL()
		_, err = c.processAPIServerService(serviceBody, httpMethod, servicesURL, sanitizedName)
		if err != nil {
			return "", err
		}
	} else {
		_, err = c.processAPIServerService(serviceBody, httpMethod, servicesURL, sanitizedName)
		if err != nil {
			return "", err
		}
	}

	// add/update api revision
	_, err = c.processAPIServerRevision(serviceBody, httpMethod, revisionsURL, sanitizedName)
	if err != nil {
		return "", err
	}

	// add/update api instance
	itemID, err = c.processAPIServerInstance(serviceBody, httpMethod, serviceInstancesURL, sanitizedName)
	if err != nil {
		return "", err
	}

	// add/update consumer instance
	if c.cfg.IsPublishToEnvironmentAndCatalogMode() {
		if !c.consumerInstanceExists(sanitizedName) {
			httpMethod = http.MethodPost
			consumerInstancesURL = c.cfg.GetAPIServerConsumerInstancesURL()
		}
		itemID, err = c.processAPIConsumerInstance(serviceBody, httpMethod, consumerInstancesURL, sanitizedName)
		if err != nil {
			return "", err
		}
	}
	return itemID, err
}

func isValidAuthPolicy(auth string) bool {
	for _, item := range ValidPolicies {
		if item == auth {
			return true
		}
	}
	return false
}

// getAPIServerConsumerInstance -
func (c *ServiceClient) getAPIServerConsumerInstance(consumerInstanceName string, queryParams map[string]string) (*APIServer, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	consumerInstanceURL := c.cfg.GetAPIServerConsumerInstancesURL() + "/" + consumerInstanceName

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

//processAPIServerService -
func (c *ServiceClient) processAPIServerService(serviceBody ServiceBody, httpMethod, servicesURL, name string) (string, error) {
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

	buffer, err := c.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}

	return c.apiServiceDeployAPI(httpMethod, servicesURL, buffer)

}

//processAPIServerRevision -
func (c *ServiceClient) processAPIServerRevision(serviceBody ServiceBody, httpMethod, revisionsURL, name string) (string, error) {
	revisionDefinition := RevisionDefinition{
		Type:  c.getRevisionDefinitionType(serviceBody),
		Value: serviceBody.Swagger,
	}
	spec := APIServiceRevisionSpec{
		APIService: name,
		Definition: revisionDefinition,
	}

	buffer, err := c.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}

	itemID, err := c.apiServiceDeployAPI(httpMethod, revisionsURL, buffer)
	if err != nil && httpMethod != http.MethodPut {
		return c.rollbackAPIService(serviceBody, name)
	}

	return itemID, err
}

//processAPIServerInstance -
func (c *ServiceClient) processAPIServerInstance(serviceBody ServiceBody, httpMethod, instancesURL, name string) (string, error) {
	endPoints, _ := c.getEndpointsBasedOnSwagger(serviceBody.Swagger, c.getRevisionDefinitionType(serviceBody))

	// reset the name here to include the stage
	spec := APIServerInstanceSpec{
		APIServiceRevision: name,
		InstanceEndPoint:   endPoints,
	}

	buffer, err := c.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}

	itemID, err := c.apiServiceDeployAPI(httpMethod, instancesURL, buffer)
	if err != nil && httpMethod != http.MethodPut {
		return c.rollbackAPIService(serviceBody, name)
	}

	return itemID, err
}

//processAPIConsumerInstance - deal with either a create or update of a consumerInstance
func (c *ServiceClient) processAPIConsumerInstance(serviceBody ServiceBody, httpMethod, instancesURL, name string) (string, error) {
	doc, err := strconv.Unquote(string(serviceBody.Documentation))
	if err != nil {
		return "", err
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
	if c.cfg.GetSubscriptionApprovalMode() == corecfg.AutoApproval {
		autoSubscribe = true
	}

	spec := v1alpha1.ConsumerInstanceSpec{
		Name:               serviceBody.NameToPush,
		ApiServiceInstance: name,
		Description:        serviceBody.Description,
		Visibility:         "RESTRICTED",
		Version:            serviceBody.Version,
		State:              PublishedState,
		Status:             "GA",
		Tags:               c.mapToTagsArray(serviceBody.Tags),
		Documentation:      doc,
		Subscription: v1alpha1.ConsumerInstanceSpecSubscription{
			Enabled:                enableSubscription,
			AutoSubscribe:          autoSubscribe,
			SubscriptionDefinition: subscriptionDefinitionName,
		},
	}

	buffer, err := c.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}

	itemID, err := c.apiServiceDeployAPI(httpMethod, instancesURL, buffer)
	if err != nil && httpMethod != http.MethodPut {
		return c.rollbackAPIService(serviceBody, name)
	}

	return itemID, err
}

// rollbackAPIService - if the process to add api/revision/instance fails, delete the api that was created
func (c *ServiceClient) rollbackAPIService(serviceBody ServiceBody, name string) (string, error) {
	spec := APIServiceSpec{}
	buffer, err := c.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}
	c.apiServiceDeployAPI(http.MethodDelete, c.cfg.DeleteAPIServerServicesURL()+"/"+name, buffer)
	return "", nil
}

// deleteConsumerInstance -
func (c *ServiceClient) deleteConsumerInstance(name string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetAPIServerConsumerInstancesURL()+"/"+name, nil)
	if err.Error() != strconv.Itoa(http.StatusNotFound) {
		return err
	}
	return nil
}

// isNewAPI -
func (c *ServiceClient) isNewAPI(serviceBody ServiceBody) bool {
	var token string
	apiName := strings.ToLower(serviceBody.APIName)
	request, err := http.NewRequest(http.MethodGet, c.cfg.GetAPIServerServicesURL()+"/"+sanitizeAPIName(serviceBody.APIName+serviceBody.Stage)+"?fields=", nil)

	if token, err = c.tokenRequester.GetToken(); err != nil {
		log.Error("Could not get token")
	}

	request.Header.Add("X-Axway-Tenant-Id", c.cfg.GetTenantID())
	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Add("Content-Type", "application/json")

	response, _ := http.DefaultClient.Do(request)
	if response.StatusCode == http.StatusNotFound {
		log.Debugf("New api found to deploy: %s", apiName)
		return true
	}
	return false
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
func (c *ServiceClient) createAPIServerBody(serviceBody ServiceBody, spec interface{}, name string) ([]byte, error) {
	attributes := make(map[string]interface{})
	attributes["externalAPIID"] = serviceBody.RestAPIID
	attributes["createdBy"] = serviceBody.CreatedBy

	newtags := c.mapToTagsArray(serviceBody.Tags)

	apiServer := APIServer{
		Name:       name,
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
			Path: gjson.Get(string(swagger), "basePath").String(),
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
	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return "12345678", nil
	}

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

	log.Debugf("HTTP response returning itemID: [%v]", itemID)
	return itemID, nil
}

// RegisterSubscriptionWebhook - Adds a new Subscription webhook. There is a single webhook
// per environment
func (c *ServiceClient) RegisterSubscriptionWebhook() error {
	// if the default is already set up, do nothing
	webhookCfg := c.cfg.GetSubscriptionApprovalWebhookConfig()
	if !webhookCfg.IsConfigured() {
		return nil
	}

	// create the secret
	err := c.createSecret()
	if err != nil {
		log.Errorf("Unable to create secret: %v", err.Error())
		return err
	}

	err = c.createWebhook()
	if err != nil {
		log.Errorf("Unable to create webhook: %v", err.Error())
		return err
	}

	return nil
}

// create the on-and-only secret for the environment
func (c *ServiceClient) createSecret() error {
	s := c.DefaultSubscriptionApprovalWebhook.GetSecret()
	spec := v1alpha1.SecretSpec{
		Data: map[string]string{DefaultSubscriptionWebhookAuthKey: base64.StdEncoding.EncodeToString([]byte(s))},
	}

	secret := v1alpha1.Secret{
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
	webhookCfg := c.cfg.GetSubscriptionApprovalWebhookConfig()
	specSecret := v1alpha1.WebhookSpecAuthSecret{
		Name: DefaultSubscriptionWebhookName,
		Key:  DefaultSubscriptionWebhookAuthKey,
	}
	authSpec := v1alpha1.WebhookSpecAuth{
		Secret: specSecret,
	}
	webSpec := v1alpha1.WebhookSpec{
		Auth:    authSpec,
		Enabled: true,
		Url:     webhookCfg.GetURL(),
		Headers: webhookCfg.GetWebhookHeaders(),
	}

	webhook := v1alpha1.Webhook{
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
