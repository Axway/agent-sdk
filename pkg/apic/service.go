package apic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
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
)

// CreateService - Creates a catalog item or API service for the definition based on the agent mode
func (c *ServiceClient) CreateService(serviceBody ServiceBody) (string, error) {
	return c.processAPIService(serviceBody)
}

// UpdateService - depending on the mode, ID might be a catalogID, a serverInstanceID, or a consumerInstanceID
func (c *ServiceClient) UpdateService(ID string, serviceBody ServiceBody) (string, error) {
	return c.processAPIService(serviceBody)
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
