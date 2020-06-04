package apic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
	log "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
	"github.com/tidwall/gjson"
)

// AddToAPIC -
func (c *ServiceClient) addCatalog(serviceBody ServiceBody) (string, error) {
	// add createdBy as a tag
	serviceBody.Tags["createdBy_"+serviceBody.CreatedBy] = ""

	serviceBody.ServiceExecution = addCatalog
	catalogID, err := c.deployCatalog(serviceBody, http.MethodPost, c.cfg.GetCatalogItemsURL())
	if err != nil {
		return "", err
	}

	log.Debugf("Catalog item with ID '%v' added", catalogID)

	if serviceBody.Image != "" {
		serviceBody.ServiceExecution = addCatalogImage
		_, err = c.deployCatalog(serviceBody, http.MethodPost, c.cfg.GetCatalogItemImageURL(catalogID))
		if err != nil {
			log.Warn("Unable to add image to the catalog item. " + err.Error())
		}
	}
	return catalogID, nil
}

func (c *ServiceClient) deployCatalog(serviceBody ServiceBody, method, url string) (string, error) {
	if !isValidAuthPolicy(serviceBody.AuthPolicy) {
		return "", fmt.Errorf("Unsupported security policy '%v' for FrontEndProxy '%s'. ", serviceBody.AuthPolicy, serviceBody.APIName)
	}

	buffer, err := c.createCatalogBody(serviceBody)
	if err != nil {
		log.Error("Error creating service item: ", err)
		return "", err
	}

	return c.catalogDeployAPI(method, url, buffer)
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
	case deleteCatalog:
		break // empty spec for delete
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

	definitionSubType, revisionPropertyKey := c.getDefinitionSubtypeAndRevisionKey(serviceBody)

	catalogProperties := []CatalogProperty{}
	if definitionSubType != Wsdl {
		catalogProperties = []CatalogProperty{
			{
				Key: "accessInfo",
				Value: CatalogPropertyValue{
					AuthPolicy: serviceBody.AuthPolicy,
					URL:        serviceBody.URL,
				},
			},
		}
	}

	newCatalogItem := CatalogItemInit{
		DefinitionType:     API,
		DefinitionSubType:  definitionSubType,
		DefinitionRevision: 1,
		Name:               serviceBody.NameToPush,
		OwningTeamID:       serviceBody.TeamID,
		Description:        serviceBody.Description,
		Properties:         catalogProperties,
		Tags:               c.mapToTagsArray(serviceBody.Tags),
		Visibility:         "RESTRICTED", // default value
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
			State:   PublishedState,
			Properties: []CatalogRevisionProperty{
				{
					Key:   "documentation",
					Value: json.RawMessage(string(serviceBody.Documentation)),
				},
				{
					Key:   revisionPropertyKey,
					Value: c.getRawMessageFromSwagger(serviceBody),
				},
			},
		},
	}

	return json.Marshal(newCatalogItem)
}

// marshal the CatalogItem -
func (c *ServiceClient) marshalCatalogItem(serviceBody ServiceBody) ([]byte, error) {

	definitionSubType, _ := c.getDefinitionSubtypeAndRevisionKey(serviceBody)

	newCatalogItem := CatalogItem{
		DefinitionType:    API,
		DefinitionSubType: definitionSubType,

		DefinitionRevision: 1,
		Name:               serviceBody.NameToPush,
		OwningTeamID:       serviceBody.TeamID,
		Description:        serviceBody.Description,
		Tags:               c.mapToTagsArray(serviceBody.Tags),
		Visibility:         "RESTRICTED",   // default value
		State:              PublishedState, //default
		LatestVersionDetails: CatalogItemRevision{
			Version: serviceBody.Version,
			State:   PublishedState,
		},
	}

	return json.Marshal(newCatalogItem)
}

// marshal the CatalogItem revision
func (c *ServiceClient) marshalCatalogItemRevision(serviceBody ServiceBody) ([]byte, error) {

	_, revisionPropertyKey := c.getDefinitionSubtypeAndRevisionKey(serviceBody)

	catalogItemRevision := CatalogItemInitRevision{
		Version: serviceBody.Version,
		State:   PublishedState,
		Properties: []CatalogRevisionProperty{
			{
				Key:   "documentation",
				Value: json.RawMessage(string(serviceBody.Documentation)),
			},
			{
				Key:   revisionPropertyKey,
				Value: c.getRawMessageFromSwagger(serviceBody),
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

func (c *ServiceClient) getDefinitionSubtypeAndRevisionKey(serviceBody ServiceBody) (definitionSubType, revisionPropertyKey string) {
	if serviceBody.ResourceType == Wsdl {
		definitionSubType = Wsdl
		revisionPropertyKey = Specification
	} else {
		oasVer := gjson.GetBytes(serviceBody.Swagger, "openapi")
		definitionSubType = SwaggerV2
		revisionPropertyKey = Swagger
		if oasVer.Exists() {
			// OAS v3
			definitionSubType = Oas3
			revisionPropertyKey = Specification
		}
	}
	return
}

func (c *ServiceClient) getRawMessageFromSwagger(serviceBody ServiceBody) (rawMsg json.RawMessage) {
	if serviceBody.ResourceType == Wsdl {
		str := base64.StdEncoding.EncodeToString(serviceBody.Swagger)
		rawMsg = json.RawMessage(strconv.Quote(str))
	} else {
		rawMsg = json.RawMessage(serviceBody.Swagger)
	}
	return
}

// UpdateCatalogItemRevisions -
func (c *ServiceClient) UpdateCatalogItemRevisions(ID string, serviceBody ServiceBody) (string, error) {
	serviceBody.ServiceExecution = updateCatalogRevision
	return c.deployCatalog(serviceBody, http.MethodPost, c.cfg.UpdateCatalogItemRevisions(ID))
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

// getCatalogItemIDForConsumerInstance -
func (c *ServiceClient) getCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	headers, err := c.createHeader()
	if err != nil {
		return "", err
	}

	params := map[string]string{
		"query": fmt.Sprintf("relationships.type==API_SERVER_CONSUMER_INSTANCE_ID;relationships.value==%s", instanceID),
	}
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetCatalogItemsURL(),
		Headers:     headers,
		QueryParams: params,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return "", err
	}
	if !(response.Code == http.StatusOK) {
		logResponseErrors(response.Body)
		return "", errors.New(strconv.Itoa(response.Code))
	}

	// the response is an array of IDs
	ids := gjson.Get(string(response.Body), "#.id")
	if !ids.Exists() {
		return "", nil
	}

	// the array should only contain 1 item,
	// since we have asked for a specific one
	catalogIDs := make([]string, 0)
	json.Unmarshal([]byte(ids.Raw), &catalogIDs)
	catalogItems := make([]CatalogItem, 0)
	if len(catalogIDs) == 0 {
		return "", errors.New("Unable to find catalogID for consumerInstance " + instanceID)
	}

	err = json.Unmarshal(response.Body, &catalogItems)
	if err != nil {
		return "", err
	}

	return catalogIDs[0], nil
}

// getConsumerInstanceForCatalogItem -
func (c *ServiceClient) getConsumerInstanceForCatalogItem(itemID string) (*APIServer, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"query": "type==API_SERVER_CONSUMER_INSTANCE_NAME",
	}
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetCatalogItemRelationshipsURL(itemID),
		Headers:     headers,
		QueryParams: params,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		logResponseErrors(response.Body)
		return nil, errors.New(strconv.Itoa(response.Code))
	}

	relationships := make([]EntityRelationship, 0)
	err = json.Unmarshal(response.Body, &relationships)
	if err != nil {
		return nil, err
	}
	if len(relationships) == 0 {
		return nil, errors.New("No relationships found")
	}

	return c.getAPIServerConsumerInstance(relationships[0].Value)
}

func isValidAuthPolicy(auth string) bool {
	for _, item := range ValidPolicies {
		if item == auth {
			return true
		}
	}
	return false
}

// updateCatalog -
func (c *ServiceClient) updateCatalog(catalogID string, serviceBody ServiceBody) (string, error) {
	serviceBody.ServiceExecution = updateCatalog
	_, err := c.deployCatalog(serviceBody, http.MethodPut, c.cfg.GetCatalogItemsURL()+"/"+catalogID)
	if err != nil {
		return "", err
	}

	if serviceBody.Image != "" {
		serviceBody.ServiceExecution = addCatalogImage
		_, err = c.deployCatalog(serviceBody, http.MethodPost, c.cfg.GetCatalogItemImageURL(catalogID))
		if err != nil {
			log.Warn("Unable to add image to the catalog item. " + err.Error())
		}
	}

	version, err := c.GetCatalogItemRevision(catalogID)
	i, err := strconv.Atoi(version)

	serviceBody.Version = strconv.Itoa(i + 1)
	_, err = c.UpdateCatalogItemRevisions(catalogID, serviceBody)
	if err != nil {
		return "", err
	}

	err = c.updateCatalogSubscription(catalogID, serviceBody)
	if err != nil {
		log.Warnf("Unable to update subscription for catalog with ID '%s'. %v", catalogID, err.Error())
	}
	return catalogID, nil
}

// updateCatalogSubscription -
func (c *ServiceClient) updateCatalogSubscription(catalogID string, serviceBody ServiceBody) error {
	// if the current state is unpublished, unsubscribe the catalog item. NOTE: despite the API docs that say the
	// value of the state is UPPER, the api returns LOWER. Make them all the same before comparing
	if strings.EqualFold(serviceBody.PubState, UnpublishedState) {
		_, err := c.unsubscribeCatalogItem(catalogID)
		if err != nil {
			return err
		}
	}
	return nil
}

// unsubscribeCatalogItem - move the catalog item to unsubscribed state
func (c *ServiceClient) unsubscribeCatalogItem(catalogItemID string) (int, error) {
	if c.cfg.IsPublishToCatalogMode() || c.cfg.IsPublishToEnvironmentAndCatalogMode() {
		subscriptions, err := c.getActiveSubscriptionsForCatalogItem(catalogItemID)
		if err != nil {
			return 0, err
		}

		for _, subscription := range subscriptions {
			// just initiate the unsubscibe, and let the poller handle finishing it all up
			log.Debugf("Unsubscribing from active subscription %s for catalog item ID %s", subscription.Name, catalogItemID)
			subscription.apicClient = c
			err = subscription.UpdateState(SubscriptionUnsubscribeInitiated)
			if err != nil {
				return len(subscriptions), err
			}
		}
		return len(subscriptions), nil
	}

	return 0, nil
}

// catalogDeployAPI -
func (c *ServiceClient) catalogDeployAPI(method, url string, buffer []byte) (string, error) {
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

	if !(response.Code == http.StatusOK || response.Code == http.StatusCreated || response.Code == http.StatusNoContent) {
		logResponseErrors(response.Body)
		return "", errors.New(strconv.Itoa(response.Code))
	}

	itemID := gjson.Get(string(response.Body), "id").String()
	return itemID, nil
}

// deleteCatalogItem -
func (c *ServiceClient) deleteCatalogItem(catalogID string, serviceBody ServiceBody) error {
	serviceBody.ServiceExecution = deleteCatalog
	_, err := c.deployCatalog(serviceBody, http.MethodDelete, c.cfg.GetCatalogItemsURL()+"/"+catalogID)
	if err != nil {
		return err
	}

	return nil
}

func (c *ServiceClient) doesCatalogItemForServiceHaveActiveSubscriptions(instanceID string) (bool, error) {
	catalogID, err := c.getCatalogItemIDForConsumerInstance(instanceID)
	if err != nil {
		return false, err
	}
	subscriptions, err := c.getActiveSubscriptionsForCatalogItem(catalogID)
	if err != nil {
		return false, err
	}
	return len(subscriptions) > 0, nil
}
