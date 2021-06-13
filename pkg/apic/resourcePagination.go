package apic

import (
	"encoding/json"
	"strconv"

	"strings"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func (c *ServiceClient) GetAPIServiceInstances(queryParams map[string]string) ([]*v1alpha1.APIServiceInstance, error) {
	resources, err := c.getAPIResources(queryParams, c.cfg.GetInstancesURL(), "")
	if err != nil {
		return nil, err
	}
	apiServiceIntances, err := v1alpha1.APIServiceInstanceFromInstanceArray(resources)
	if err != nil {
		return nil, err
	}

	return apiServiceIntances, nil
}

func (c *ServiceClient) GetAPIServiceRevisions(queryParams map[string]string, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	resources, err := c.getAPIResources(queryParams, c.cfg.GetInstancesURL(), "")
	if err != nil {
		return nil, err
	}
	apiServiceIntances, err := v1alpha1.APIServiceRevisionFromInstanceArray(resources)
	if err != nil {
		return nil, err
	}

	filteredAPIRevisions := make([]*v1alpha1.APIServiceRevision, 0)

	//create array and filter by stage name. Check the stage name as this does not apply for v7
	if stage != "" {
		for _, apiServer := range apiServiceIntances {
			if strings.Contains(strings.ToLower(apiServer.Name), strings.ToLower(stage)) {
				filteredAPIRevisions = append(filteredAPIRevisions, apiServer)
			}
		}
	} else {
		filteredAPIRevisions = apiServiceIntances
	}

	return filteredAPIRevisions, nil
}

func (c *ServiceClient) getAPIResources(queryParams map[string]string, URL, stage string) ([]*apiv1.ResourceInstance, error) {
	morePages := true
	page := 1

	resourceInstance := make([]*apiv1.ResourceInstance, 0)

	for morePages {
		query := map[string]string{
			"page":     strconv.Itoa(page),
			"pageSize": strconv.Itoa(apiServerPageSize),
		}

		// Add query params for getting revisions for the service and use the latest one as last reference
		for key, value := range queryParams {
			query[key] = value
		}

		response, err := c.ExecuteAPI(coreapi.GET, URL, query, nil)

		if err != nil {
			log.Debugf("Error while retrieving ResourceInstance: %s", err.Error())
			return nil, err
		}

		resourceInstancePage := make([]*apiv1.ResourceInstance, 0)
		json.Unmarshal(response, &resourceInstancePage)

		resourceInstance = append(resourceInstance, resourceInstancePage...)

		if len(resourceInstancePage) < apiServerPageSize {
			morePages = false
		}
		page++
	}

	return resourceInstance, nil
}
