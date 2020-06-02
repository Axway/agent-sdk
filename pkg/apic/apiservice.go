package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
	log "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/wsdl"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tidwall/gjson"
)

type apiService struct {
	serviceClient *ServiceClient
}

func newAPIService(serviceClient *ServiceClient) *apiService {
	return &apiService{
		serviceClient: serviceClient,
	}
}

// processAPIService - This function will add or update the api service
// If the api doesn't exist, it will add the service, revision, and instance.
// If the api does exist, it will update the revision and instance
func (a *apiService) processAPIService(serviceBody ServiceBody) (string, error) {
	if !isValidAuthPolicy(serviceBody.AuthPolicy) {
		return "", fmt.Errorf("Unsupported security policy '%v'. ", serviceBody.AuthPolicy)
	}

	itemID := ""
	var err error
	httpMethod := http.MethodPut
	sanitizedName := sanitizeAPIName(serviceBody.APIName + serviceBody.Stage)
	servicesURL := a.serviceClient.cfg.GetAPIServerServicesURL() + "/" + sanitizedName
	revisionsURL := a.serviceClient.cfg.GetAPIServerServicesRevisionsURL() + "/" + sanitizedName
	serviceInstancesURL := a.serviceClient.cfg.GetAPIServerServicesInstancesURL() + "/" + sanitizedName
	consumerInstancesURL := a.serviceClient.cfg.GetAPIServerConsumerInstancesURL() + "/" + sanitizedName

	// Verify if the api already exists
	if a.isNewAPI(serviceBody) {
		// add api
		httpMethod = http.MethodPost
		servicesURL := a.serviceClient.cfg.GetAPIServerServicesURL()
		revisionsURL = a.serviceClient.cfg.GetAPIServerServicesRevisionsURL()
		serviceInstancesURL = a.serviceClient.cfg.GetAPIServerServicesInstancesURL()
		consumerInstancesURL = a.serviceClient.cfg.GetAPIServerConsumerInstancesURL()
		_, err = a.processAPIServerService(serviceBody, httpMethod, servicesURL, sanitizedName)
		if err != nil {
			return "", err
		}
	} else {
		_, err = a.processAPIServerService(serviceBody, httpMethod, servicesURL, sanitizedName)
		if err != nil {
			return "", err
		}
	}

	// add/update api revision
	_, err = a.processAPIServerRevision(serviceBody, httpMethod, revisionsURL, sanitizedName)
	if err != nil {
		return "", err
	}

	// add/update api instance
	itemID, err = a.processAPIServerInstance(serviceBody, httpMethod, serviceInstancesURL, sanitizedName)
	if err != nil {
		return "", err
	}

	// add/update consumer instance
	if a.serviceClient.cfg.IsPublishToEnvironmentAndCatalogMode() {
		itemID, err = a.processAPIConsumerInstance(serviceBody, httpMethod, consumerInstancesURL, sanitizedName)
		if err != nil {
			return "", err
		}
	}

	return itemID, err
}

//processAPIServerService -
func (a *apiService) processAPIServerService(serviceBody ServiceBody, httpMethod, servicesURL, name string) (string, error) {
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

	buffer, err := a.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}

	return a.apiServiceDeployAPI(httpMethod, servicesURL, buffer)

}

//processAPIServerRevision -
func (a *apiService) processAPIServerRevision(serviceBody ServiceBody, httpMethod, revisionsURL, name string) (string, error) {
	revisionDefinition := RevisionDefinition{
		Type:  a.getRevisionDefinitionType(serviceBody),
		Value: serviceBody.Swagger,
	}
	spec := APIServiceRevisionSpec{
		APIService: name,
		Definition: revisionDefinition,
	}

	buffer, err := a.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}

	itemID, err := a.apiServiceDeployAPI(httpMethod, revisionsURL, buffer)
	if err != nil && httpMethod != http.MethodPut {
		return a.rollbackAPIService(serviceBody, name)
	}

	return itemID, err
}

//processAPIServerInstance -
func (a *apiService) processAPIServerInstance(serviceBody ServiceBody, httpMethod, instancesURL, name string) (string, error) {
	endPoints, _ := a.getEndpointsBasedOnSwagger(serviceBody.Swagger, a.getRevisionDefinitionType(serviceBody))

	// reset the name here to include the stage
	spec := APIServerInstanceSpec{
		APIServiceRevision: name,
		InstanceEndPoint:   endPoints,
	}

	buffer, err := a.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}

	itemID, err := a.apiServiceDeployAPI(httpMethod, instancesURL, buffer)
	if err != nil && httpMethod != http.MethodPut {
		return a.rollbackAPIService(serviceBody, name)
	}

	return itemID, err
}

//processAPIConsumerInstance -
func (a *apiService) processAPIConsumerInstance(serviceBody ServiceBody, httpMethod, instancesURL, name string) (string, error) {
	doc, err := strconv.Unquote(string(serviceBody.Documentation))
	if err != nil {
		return "", err
	}
	spec := ConsumerInstanceSpec{
		Name:               serviceBody.NameToPush,
		APIServiceInstance: name,
		Description:        serviceBody.Description,
		Visibility:         "RESTRICTED",
		Version:            serviceBody.Version,
		State:              PublishedState,
		Status:             "GA",
		Tags:               a.serviceClient.mapToTagsArray(serviceBody.Tags),
		Documentation:      doc,
		Subscription: &APIServiceSubscription{
			Enabled:                true,
			AutoSubscribe:          true,
			SubscriptionDefinition: a.serviceClient.cfg.GetEnvironmentName() + "." + "authsubscription",
		},
	}

	buffer, err := a.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}

	itemID, err := a.apiServiceDeployAPI(httpMethod, instancesURL, buffer)
	if err != nil && httpMethod != http.MethodPut {
		return a.rollbackAPIService(serviceBody, name)
	}

	return itemID, err
}

// rollbackAPIService - if the process to add api/revision/instance fails, delete the api that was created
func (a *apiService) rollbackAPIService(serviceBody ServiceBody, name string) (string, error) {
	spec := APIServiceSpec{}
	buffer, err := a.createAPIServerBody(serviceBody, spec, name)
	if err != nil {
		return "", err
	}
	a.apiServiceDeployAPI(http.MethodDelete, a.serviceClient.cfg.DeleteAPIServerServicesURL()+"/"+name, buffer)
	return "", nil
}

// isNewAPI -
func (a *apiService) isNewAPI(serviceBody ServiceBody) bool {
	var token string
	apiName := strings.ToLower(serviceBody.APIName)
	request, err := http.NewRequest("GET", a.serviceClient.cfg.GetAPIServerServicesURL()+"/"+sanitizeAPIName(serviceBody.APIName+serviceBody.Stage), nil)

	if token, err = a.serviceClient.tokenRequester.GetToken(); err != nil {
		log.Error("Could not get token")
	}

	request.Header.Add("X-Axway-Tenant-Id", a.serviceClient.cfg.GetTenantID())
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
func (a *apiService) getRevisionDefinitionType(serviceBody ServiceBody) string {
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
func (a *apiService) createAPIServerBody(serviceBody ServiceBody, spec interface{}, name string) ([]byte, error) {
	attributes := make(map[string]interface{})
	attributes["externalAPIID"] = serviceBody.RestAPIID
	attributes["createdBy"] = serviceBody.CreatedBy

	newtags := a.serviceClient.mapToTagsArray(serviceBody.Tags)

	apiServer := APIServer{
		Name:       name,
		Title:      serviceBody.NameToPush,
		Attributes: attributes,
		Spec:       spec,
		Tags:       newtags,
	}

	return json.Marshal(apiServer)
}

func (a *apiService) getEndpointsBasedOnSwagger(swagger []byte, revisionDefinitionType string) ([]EndPoint, error) {
	switch revisionDefinitionType {
	case Wsdl:
		return a.getWsdlEndpoints(swagger)
	case Oas2:
		return a.getOas2Endpoints(swagger)
	case Oas3:
		return a.getOas3Endpoints(swagger)
	}

	return nil, fmt.Errorf("Unable to get endpoints from swagger; invalid definition type: %v", revisionDefinitionType)
}

func (a *apiService) getWsdlEndpoints(swagger []byte) ([]EndPoint, error) {
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

func (a *apiService) getOas2Endpoints(swagger []byte) ([]EndPoint, error) {
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

func (a *apiService) getOas3Endpoints(swagger []byte) ([]EndPoint, error) {
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
			defaultURL, allURLs, err = a.handleURLSubstitutions(server, allURLs)
			if err != nil {
				return nil, err
			}
		}

		parsedEndPoints, err := a.parseURLsIntoEndpoints(defaultURL, allURLs)
		if err != nil {
			return nil, err
		}
		endPoints = append(endPoints, parsedEndPoints...)
	}

	return endPoints, nil
}

func (a *apiService) handleURLSubstitutions(server *openapi3.Server, allURLs []string) (string, []string, error) {
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

func (a *apiService) parseURLsIntoEndpoints(defaultURL string, allURLs []string) ([]EndPoint, error) {
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

// apiServiceDeployAPI -
func (a *apiService) apiServiceDeployAPI(method, url string, buffer []byte) (string, error) {
	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return "12345678", nil
	}

	headers, err := a.serviceClient.createHeader()
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
	response, err := a.serviceClient.apiClient.Send(request)
	if err != nil {
		return "", err
	}
	//  Check to see if rollback was processed
	if method == http.MethodDelete && response.Code == http.StatusNoContent {
		log.Error("Rollback API service.  API has been removed.")
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
