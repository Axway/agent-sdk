package apic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
)

type actionType int

const (
	addAPI    actionType = iota
	updateAPI            = iota
	deleteAPI            = iota
)

type serviceExecution int

const (
	addAPIServerSpec serviceExecution = iota + 1
	addAPIServerRevisionSpec
	addAPIServerInstanceSpec
	deleteAPIServerSpec
	addCatalog
	addCatalogImage
	updateCatalog
	deleteCatalog
	updateCatalogRevision
	getCatalogItem
)

// CreateService - Creates a catalog item or API service for the definition based on the agent mode
func (c *ServiceClient) CreateService(serviceBody ServiceBody) (string, error) {
	if c.cfg.IsPublishToEnvironmentMode() {
		return c.processAPIService(serviceBody)
	}
	return c.addCatalog(serviceBody)
}

// UpdateService - depending on the mode, ID might be a catalogID, a serverInstanceID, or a consumerInstanceID
func (c *ServiceClient) UpdateService(ID string, serviceBody ServiceBody) (string, error) {
	if c.cfg.IsPublishToEnvironmentMode() {
		return c.processAPIService(serviceBody)
	}
	return c.updateCatalog(ID, serviceBody)
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

