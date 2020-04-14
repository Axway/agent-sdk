package apic

import (
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
	log "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/wsdl"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tidwall/gjson"
)

// processAPIService - This function will add or update the api service
// If the api doesn't exist, it will add the service, revision, and instance.
// If the api does exist, it will update the revision and instance
func (c *ServiceClient) processAPIService(serviceBody ServiceBody) (string, error) {
	itemID := ""
	httpMethod := http.MethodPut
	revisionsURL := c.cfg.GetAPIServerServicesRevisionsURL() + "/" + sanitizeAPIName(serviceBody.APIName)
	instancesURL := c.cfg.GetAPIServerServicesInstancesURL() + "/" + sanitizeAPIName(serviceBody.APIName)

	// Verify if the api already exists
	if c.isNewAPI(serviceBody) {
		// add api
		httpMethod = http.MethodPost
		revisionsURL = c.cfg.GetAPIServerServicesRevisionsURL()
		instancesURL = c.cfg.GetAPIServerServicesInstancesURL()
		serviceBody.ServiceExecution = addAPIServerSpec
		itemID, err := c.deployService(serviceBody, httpMethod, c.cfg.GetAPIServerServicesURL())
		if err != nil {
			return itemID, err
		}
	}

	// add/update api revision
	serviceBody.ServiceExecution = addAPIServerRevisionSpec
	itemID, err := c.deployService(serviceBody, httpMethod, revisionsURL)
	if err != nil {
		log.Errorf("Error adding API revision for API %v", serviceBody.NameToPush)
		if httpMethod == http.MethodPost {
			return c.rollbackAPIService(serviceBody)
		}
	}

	// add/update api instance
	serviceBody.ServiceExecution = addAPIServerInstanceSpec
	itemID, err = c.deployService(serviceBody, httpMethod, instancesURL)
	if err != nil {
		log.Errorf("Error adding API instance for API %v", serviceBody.NameToPush)
		if httpMethod == http.MethodPost {
			return c.rollbackAPIService(serviceBody)
		}
	}

	return itemID, err
}

// rollbackAPIService - if the process to add api/revision/instance fails, delete the api that was created
func (c *ServiceClient) rollbackAPIService(serviceBody ServiceBody) (string, error) {
	// rollback and remove the API service
	serviceBody.ServiceExecution = deleteAPIServerSpec
	return c.deployService(serviceBody, http.MethodDelete, c.cfg.DeleteAPIServerServicesURL()+"/"+sanitizeAPIName(serviceBody.APIName))
}

// IsNewAPI -
func (c *ServiceClient) isNewAPI(serviceBody ServiceBody) bool {
	var token string
	apiName := strings.ToLower(serviceBody.APIName)
	request, err := http.NewRequest("GET", c.cfg.GetAPIServerServicesURL()+"/"+apiName, nil)

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

func (c *ServiceClient) deployService(serviceBody ServiceBody, method, url string) (string, error) {
	if !isValidAuthPolicy(serviceBody.AuthPolicy) {
		return "", fmt.Errorf("Unsupported security policy '%v'. ", serviceBody.AuthPolicy)
	}
	buffer, err := c.createAPIServerBody(serviceBody)
	if err != nil {
		log.Error("Error creating service item: ", err)
		return "", err
	}

	return c.apiServiceDeployAPI(method, url, buffer)
}

// createAPIServerBody - This function is being used by both the api server creation and api server revision creation
func (c *ServiceClient) createAPIServerBody(serviceBody ServiceBody) ([]byte, error) {
	attributes := make(map[string]interface{})
	attributes["externalAPIID"] = serviceBody.RestAPIID
	attributes["createdBy"] = serviceBody.CreatedBy

	// spec needs to adhere to environment schema
	var spec interface{}
	name := sanitizeAPIName(serviceBody.APIName)

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

	switch serviceBody.ServiceExecution {
	case addAPIServerSpec:
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
	case deleteAPIServerSpec:
		spec = APIServiceSpec{}
		return nil, nil
	case addAPIServerRevisionSpec:
		revisionDefinition := RevisionDefinition{
			Type:  revisionDefinitionType,
			Value: serviceBody.Swagger,
		}
		spec = APIServiceRevisionSpec{
			APIService: name,
			Definition: revisionDefinition,
		}
		// reset the name here to include the stage
		name = sanitizeAPIName(serviceBody.APIName + serviceBody.Stage)
	case addAPIServerInstanceSpec:
		endPoints, _ := c.getEndpointsBasedOnSwagger(serviceBody.Swagger, revisionDefinitionType)

		// reset the name here to include the stage
		name = sanitizeAPIName(serviceBody.APIName + serviceBody.Stage)
		spec = APIServerInstanceSpec{
			APIServiceRevision: name,
			InstanceEndPoint:   endPoints,
		}
	default:
		return nil, errors.New("Error getting execution service -- not set")
	}

	newtags := c.mapToTagsArray(serviceBody.Tags)

	apiServerService := APIServer{
		Name:       name,
		Title:      serviceBody.NameToPush,
		Attributes: attributes,
		Spec:       spec,
		Tags:       newtags,
	}

	return json.Marshal(apiServerService)
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

		endPoint := EndPoint{
			Host:     host,
			Port:     port,
			Protocol: protocol,
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
	for _, protocol := range schemes {
		endPoint := EndPoint{
			Host:     host,
			Port:     port,
			Protocol: protocol,
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

		if server.Variables != nil {
			defaultURL = server.URL
			// Handle substitutions
			for serverKey, serverVar := range server.Variables {
				newURLs := []string{}
				if serverVar.Default == nil {
					err := fmt.Errorf("Server variable in OAS3 %s does not have a default value, spec not valid", serverKey)
					log.Errorf(err.Error())
					return nil, err
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
		}

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

			endPoint := EndPoint{
				Host:     urlObj.Hostname(),
				Port:     port,
				Protocol: urlObj.Scheme,
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
	}

	return endPoints, nil
}

// Sanitize name to be path friendly and follow this regex: ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*
func sanitizeAPIName(name string) string {
	// convert all letters to lower first
	newName := strings.ToLower(name)
	// fmt.Println(name)

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
