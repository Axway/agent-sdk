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
	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	log "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tidwall/gjson"
)

type actionType int

const (
	addAPI    actionType = iota
	updateAPI            = iota
	deleteAPI            = iota
)

type serviceExecution int

const unpublishedState = "UNPUBLISHED"
const publishedState = "PUBLISHED"

const (
	addAPIServerSpec serviceExecution = iota + 1
	addAPIServerRevisionSpec
	addAPIServerInstanceSpec
	deleteAPIServerSpec
	addCatalog
	addCatalogImage
	updateCatalog
	updateCatalogRevision
	getCatalogItem
)

func (c *ServiceClient) deployService(serviceBody ServiceBody, method, url string) (string, error) {
	buffer, err := c.marshalServiceBody(serviceBody)
	if err != nil {
		log.Error("Error creating service item: ", err)
		return "", err
	}

	return c.deployAPI(method, url, buffer)
}

// CreateService - Creates a catalog item or API service for the definition based on the agent mode
func (c *ServiceClient) CreateService(serviceBody ServiceBody) (string, error) {
	if c.cfg.GetAgentMode() == corecfg.Connected {
		return c.addAPICService(serviceBody)
	}
	return c.addCatalog(serviceBody)
}

// AddToAPICServer -
func (c *ServiceClient) addAPICService(serviceBody ServiceBody) (string, error) {
	itemID := ""
	// Verify if the api already exists
	if c.isNewAPI(serviceBody) {
		// add api
		serviceBody.ServiceExecution = addAPIServerSpec
		itemID, err := c.deployService(serviceBody, http.MethodPost, c.cfg.GetAPIServerServicesURL())
		if err != nil {
			return itemID, err
		}
	}

	// add api revision
	serviceBody.ServiceExecution = addAPIServerRevisionSpec
	itemID, err := c.deployService(serviceBody, http.MethodPost, c.cfg.GetAPIServerServicesRevisionsURL())
	if err != nil {
		log.Errorf("Error adding API revision for API %v", serviceBody.NameToPush)
		return c.rollbackAPIService(serviceBody)
	}

	// add api instance
	serviceBody.ServiceExecution = addAPIServerInstanceSpec
	itemID, err = c.deployService(serviceBody, http.MethodPost, c.cfg.GetAPIServerServicesInstancesURL())
	if err != nil {
		log.Errorf("Error adding API instance for API %v", serviceBody.NameToPush)
		return c.rollbackAPIService(serviceBody)
	}

	return itemID, err
}

// rollbackAPIService - if the process to add api/revision/instance fails, delete the api that was created
func (c *ServiceClient) rollbackAPIService(serviceBody ServiceBody) (string, error) {
	// rollback and remove the API service
	serviceBody.ServiceExecution = deleteAPIServerSpec
	return c.deployService(serviceBody, http.MethodDelete, c.cfg.DeleteAPIServerServicesURL()+"/"+sanitizeAPIName(serviceBody.APIName))
}

// AddToAPIC -
func (c *ServiceClient) addCatalog(serviceBody ServiceBody) (string, error) {
	serviceBody.ServiceExecution = addCatalog
	itemID, err := c.deployService(serviceBody, http.MethodPost, c.cfg.GetCatalogItemsURL())
	if err != nil {
		return "", err
	}
	if serviceBody.Image != "" {
		serviceBody.ServiceExecution = addCatalogImage
		_, err = c.deployService(serviceBody, http.MethodPost, c.cfg.GetCatalogItemImageURL(itemID))
		if err != nil {
			log.Warn("Unable to add image to the catalog item. " + err.Error())
		}
	}
	return itemID, nil
}

// UpdateService -
func (c *ServiceClient) UpdateService(ID string, serviceBody ServiceBody) (string, error) {
	serviceBody.ServiceExecution = updateCatalog
	_, err := c.deployService(serviceBody, http.MethodPut, c.cfg.GetCatalogItemsURL()+"/"+ID)
	if err != nil {
		return "", err
	}

	version, err := c.GetCatalogItemRevision(ID)
	i, err := strconv.Atoi(version)

	serviceBody.Version = strconv.Itoa(i + 1)
	_, err = c.UpdateCatalogItemRevisions(ID, serviceBody)
	if err != nil {
		return "", err
	}

	return ID, nil
}

// UpdateCatalogItemRevisions -
func (c *ServiceClient) UpdateCatalogItemRevisions(ID string, serviceBody ServiceBody) (string, error) {
	serviceBody.ServiceExecution = updateCatalogRevision
	return c.deployService(serviceBody, http.MethodPost, c.cfg.UpdateCatalogItemRevisions(ID))
}

// GetCatalogItemRevision -
func (c *ServiceClient) GetCatalogItemRevision(ID string) (string, error) {
	headers, err := c.createHeader()
	if err != nil {
		return "", err
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.cfg.GetCatalogItemByID(ID),
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return "", err
	}
	if !(response.Code == http.StatusOK) {
		logResponseErrors(response.Body)
		return "", errors.New(strconv.Itoa(response.Code))
	}

	revisions := gjson.Get(string(response.Body), "availableRevisions")
	availableRevisions := make([]int, 0)
	json.Unmarshal([]byte(revisions.Raw), &availableRevisions)
	revision := availableRevisions[len(availableRevisions)-1] // get the latest revsions
	return strconv.Itoa(revision), nil
}

// CreateService -
func (c *ServiceClient) marshalServiceBody(serviceBody ServiceBody) ([]byte, error) {
	if !isValidAuthPolicy(serviceBody.AuthPolicy) {
		return nil, fmt.Errorf("Unsupported security policy '%v'. ", serviceBody.AuthPolicy)
	}
	if serviceBody.AgentMode == corecfg.Connected {
		return c.createAPIServerBody(serviceBody)
	}
	return c.createCatalogBody(serviceBody)
}

func (c *ServiceClient) createCatalogBody(serviceBody ServiceBody) ([]byte, error) {
	var spec []byte
	var err error
	switch serviceBody.ServiceExecution {
	case addCatalog:
		spec, err = c.marshalCatalogItemInit(serviceBody)
	case addCatalogImage:
		spec, err = c.marshalCatalogItemImage(serviceBody)
	case updateCatalog:
		spec, err = c.marshalCatalogItem(serviceBody)
	case updateCatalogRevision:
		spec, err = c.marshalCatalogItemRevision(serviceBody)
	default:
		return nil, errors.New("Invalid catalog operation")
	}
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func (c *ServiceClient) marshalCatalogItemInit(serviceBody ServiceBody) ([]byte, error) {
	enableSubscription := (serviceBody.AuthPolicy != Passthrough)

	// assume that we use the default schema unless it one is enabled and registered
	subSchema := c.DefaultSubscriptionSchema
	if enableSubscription {
		if c.RegisteredSubscriptionSchema != nil {
			subSchema = c.RegisteredSubscriptionSchema
		} else {
			enableSubscription = false
		}
	}

	catalogSubscriptionSchema, err := subSchema.rawJSON()
	if err != nil {
		return nil, err
	}

	oasVer := gjson.GetBytes(serviceBody.Swagger, "openapi")
	definitionSubType := "swaggerv2"
	revisionPropertyKey := "swagger"
	if oasVer.Exists() {
		// OAS v3
		definitionSubType = "oas3"
		revisionPropertyKey = "specification"
	}

	newCatalogItem := CatalogItemInit{
		DefinitionType:     "API",
		DefinitionSubType:  definitionSubType,
		DefinitionRevision: 1,
		Name:               serviceBody.NameToPush,
		OwningTeamID:       serviceBody.TeamID,
		Description:        serviceBody.Description,
		Properties: []CatalogProperty{
			{
				Key: "accessInfo",
				Value: CatalogPropertyValue{
					AuthPolicy: serviceBody.AuthPolicy,
					URL:        serviceBody.URL,
				},
			},
		},

		Tags:       c.mapToTagsArray(serviceBody.Tags),
		Visibility: "RESTRICTED", // default value
		Subscription: CatalogSubscription{
			Enabled:         enableSubscription,
			AutoSubscribe:   true,
			AutoUnsubscribe: false,
			Properties: []CatalogRevisionProperty{{
				Key:   "profile",
				Value: catalogSubscriptionSchema,
			}},
		},
		Revision: CatalogItemInitRevision{
			Version: serviceBody.Version,
			State:   unpublishedState,
			Properties: []CatalogRevisionProperty{
				{
					Key:   "documentation",
					Value: json.RawMessage(string(serviceBody.Documentation)),
				},
				{
					Key:   revisionPropertyKey,
					Value: json.RawMessage(serviceBody.Swagger),
				},
			},
		},
	}

	return json.Marshal(newCatalogItem)
}

func isValidAuthPolicy(auth string) bool {
	for _, item := range ValidPolicies {
		if item == auth {
			return true
		}
	}
	return false
}

func (c *ServiceClient) getEndpointsBasedOnSwagger(swagger []byte, revisionDefinitionType string) ([]EndPoint, error) {
	endPoints := []EndPoint{}

	switch revisionDefinitionType {
	case "oas2":
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
	case "oas3":
		openAPI, _ := openapi3.NewSwaggerLoader().LoadSwaggerFromData(swagger)

		for _, server := range openAPI.Servers {
			urlObj, err := url.Parse(server.URL)
			if err != nil {
				err := fmt.Errorf("Could not parse url: %s", server.URL)
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
			endPoints = append(endPoints, endPoint)
		}
	}

	return endPoints, nil
}

// createAPIServerBody - This function is being used by both the api server creation and api server revision creation
func (c *ServiceClient) createAPIServerBody(serviceBody ServiceBody) ([]byte, error) {
	attributes := make(map[string]interface{})
	attributes["externalAPIID"] = serviceBody.RestAPIID

	// spec needs to adhere to environment schema
	var spec interface{}
	name := sanitizeAPIName(serviceBody.APIName)

	oasVer := gjson.GetBytes(serviceBody.Swagger, "openapi")
	revisionDefinitionType := "oas2"
	if oasVer.Exists() {
		// OAS v3
		revisionDefinitionType = "oas3"
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

// CreateCatalogItemBodyForUpdate -
func (c *ServiceClient) marshalCatalogItem(serviceBody ServiceBody) ([]byte, error) {
	newCatalogItem := CatalogItem{
		DefinitionType:    "API",
		DefinitionSubType: "swaggerv2",

		DefinitionRevision: 1,
		Name:               serviceBody.NameToPush,
		OwningTeamID:       serviceBody.TeamID,
		Description:        serviceBody.Description,
		Tags:               c.mapToTagsArray(serviceBody.Tags),
		Visibility:         "RESTRICTED",     // default value
		State:              unpublishedState, //default
		LatestVersionDetails: CatalogItemRevision{
			Version: serviceBody.Version,
			State:   publishedState,
		},
	}

	return json.Marshal(newCatalogItem)
}

// CreateCatalogItemBodyForRevision -
func (c *ServiceClient) marshalCatalogItemRevision(serviceBody ServiceBody) ([]byte, error) {

	catalogItemRevision := CatalogItemInitRevision{
		Version: serviceBody.Version,
		State:   unpublishedState,
		Properties: []CatalogRevisionProperty{
			{
				Key:   "documentation",
				Value: json.RawMessage(string(serviceBody.Documentation)),
			},
			{
				Key:   "swagger",
				Value: json.RawMessage(serviceBody.Swagger),
			},
		},
	}

	return json.Marshal(catalogItemRevision)
}

// marshals the catalog image body
func (c *ServiceClient) marshalCatalogItemImage(serviceBody ServiceBody) ([]byte, error) {
	catalogImage := CatalogItemImage{
		DataType:      serviceBody.ImageContentType,
		Base64Content: serviceBody.Image,
	}
	return json.Marshal(catalogImage)
}

// getCatalogItemAPIServerInfoProperty -
func (c *ServiceClient) getCatalogItemAPIServerInfoProperty(catalogID string) (*APIServerInfo, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	apiServerInfoURL := c.cfg.GetCatalogItemsURL() + "/" + catalogID + "/properties/apiServerInfo"

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     apiServerInfoURL,
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		logResponseErrors(response.Body)
		return nil, errors.New(strconv.Itoa(response.Code))
	}

	apiserverInfo := new(APIServerInfo)
	json.Unmarshal(response.Body, apiserverInfo)
	return apiserverInfo, nil
}

// getAPIServerConsumerInstance -
func (c *ServiceClient) getAPIServerConsumerInstance(consumerInstanceName string) (*APIServer, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	consumerInstanceURL := c.cfg.GetAPIServerConsumerInstancesURL() + "/" + consumerInstanceName

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     consumerInstanceURL,
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		logResponseErrors(response.Body)
		return nil, errors.New(strconv.Itoa(response.Code))
	}
	consumerInstance := new(APIServer)
	json.Unmarshal(response.Body, consumerInstance)
	return consumerInstance, nil
}
