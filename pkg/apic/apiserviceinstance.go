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

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	v1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	log "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/wsdl"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tidwall/gjson"
)

func (c *ServiceClient) buildAPIServiceInstanceSpec(serviceBody *ServiceBody) v1alpha1.ApiServiceInstanceSpec {
	endPoints, _ := c.getEndpointsBasedOnSwagger(serviceBody.Swagger, c.getRevisionDefinitionType(*serviceBody))
	return v1alpha1.ApiServiceInstanceSpec{
		ApiServiceRevision: serviceBody.serviceContext.currentRevision,
		Endpoint:           endPoints,
	}
}

func (c *ServiceClient) buildAPIServiceInstanceResource(serviceBody *ServiceBody, instanceName string) *v1alpha1.APIServiceInstance {
	return &v1alpha1.APIServiceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceInstanceGVK(),
			Name:             instanceName,
			Title:            serviceBody.NameToPush,
			Attributes:       c.buildAPIResourceAttributes(serviceBody, nil, false),
			Tags:             c.mapToTagsArray(serviceBody.Tags),
		},
		Spec: c.buildAPIServiceInstanceSpec(serviceBody),
	}
}

//processInstance -
func (c *ServiceClient) processInstance(serviceBody *ServiceBody) error {
	instanceName := serviceBody.serviceContext.currentRevision

	httpMethod := http.MethodPost
	instanceURL := c.cfg.GetInstancesURL()
	serviceBody.serviceContext.instanceAction = serviceBody.serviceContext.revisionAction
	if serviceBody.serviceContext.revisionAction == updateAPI {
		httpMethod = http.MethodPut
		instanceURL += "/" + instanceName
	}

	apiInstance := c.buildAPIServiceInstanceResource(serviceBody, instanceName)
	buffer, err := json.Marshal(apiInstance)
	if err != nil {
		return err
	}

	_, err = c.apiServiceDeployAPI(httpMethod, instanceURL, buffer)
	if err != nil {
		if serviceBody.serviceContext.serviceAction == addAPI {
			_, err = c.rollbackAPIService(*serviceBody, serviceBody.serviceContext.serviceName)
		}
	} else {
		serviceBody.serviceContext.currentInstance = instanceName
	}

	return err
}

// getAPIServiceInstanceByName - Returns the API service instance for specified name
func (c *ServiceClient) getAPIServiceInstanceByName(instanceName string) (*v1alpha1.APIServiceInstance, error) {
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
			return nil, errors.New(strconv.Itoa(response.Code))
		}
		return nil, nil
	}
	apiInstance := new(v1alpha1.APIServiceInstance)
	json.Unmarshal(response.Body, apiInstance)
	return apiInstance, nil
}

func (c *ServiceClient) getEndpointsBasedOnSwagger(swagger []byte, revisionDefinitionType string) ([]v1alpha1.ApiServiceInstanceSpecEndpoint, error) {
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

func (c *ServiceClient) getWsdlEndpoints(swagger []byte) ([]v1alpha1.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := []v1alpha1.ApiServiceInstanceSpecEndpoint{}
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

		endPoint := v1alpha1.ApiServiceInstanceSpecEndpoint{
			Host:     host,
			Port:     int32(port),
			Protocol: protocol,
			Routing: v1alpha1.ApiServiceInstanceSpecRouting{
				BasePath: fixed.Path,
			},
		}
		if !contains(endPoints, endPoint) {
			endPoints = append(endPoints, endPoint)
		}
	}

	return endPoints, nil
}

func contains(endpts []v1alpha1.ApiServiceInstanceSpecEndpoint, endpt v1alpha1.ApiServiceInstanceSpecEndpoint) bool {
	for _, pt := range endpts {
		if pt == endpt {
			return true
		}
	}
	return false
}
func (c *ServiceClient) getOas2Endpoints(swagger []byte) ([]v1alpha1.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := []v1alpha1.ApiServiceInstanceSpecEndpoint{}
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
		endPoint := v1alpha1.ApiServiceInstanceSpecEndpoint{
			Host:     host,
			Port:     int32(port),
			Protocol: protocol,
			Routing: v1alpha1.ApiServiceInstanceSpecRouting{
				BasePath: gjson.Get(string(swagger), "basePath").String(),
			},
		}
		endPoints = append(endPoints, endPoint)
	}

	return endPoints, nil
}

func (c *ServiceClient) getOas3Endpoints(swagger []byte) ([]v1alpha1.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := []v1alpha1.ApiServiceInstanceSpecEndpoint{}
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

func (c *ServiceClient) parseURLsIntoEndpoints(defaultURL string, allURLs []string) ([]v1alpha1.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := []v1alpha1.ApiServiceInstanceSpecEndpoint{}
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

		endPoint := v1alpha1.ApiServiceInstanceSpecEndpoint{
			Host:     urlObj.Hostname(),
			Port:     int32(port),
			Protocol: urlObj.Scheme,
			Routing: v1alpha1.ApiServiceInstanceSpecRouting{
				BasePath: urlObj.Path,
			},
		}

		// If the URL is the default URL put it at the front of the array
		if urlStr == defaultURL {
			newEndPoints := []v1alpha1.ApiServiceInstanceSpecEndpoint{endPoint}
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
