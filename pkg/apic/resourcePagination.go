package apic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// GetAPIServiceRevisions - management.APIServiceRevision
func (c *ServiceClient) GetAPIServiceRevisions(queryParams map[string]string, URL, stage string) ([]*management.APIServiceRevision, error) {
	resources, err := c.GetAPIV1ResourceInstances(queryParams, URL)
	if err != nil {
		return nil, err
	}
	apiServiceInstances, err := management.APIServiceRevisionFromInstanceArray(resources)
	if err != nil {
		return nil, err
	}

	filteredAPIRevisions := make([]*management.APIServiceRevision, 0)

	// create array and filter by stage name. Check the stage name as this does not apply for v7
	if stage != "" {
		for _, apiServer := range apiServiceInstances {
			if strings.Contains(strings.ToLower(apiServer.Name), strings.ToLower(fmt.Sprintf("%s.", stage))) {
				filteredAPIRevisions = append(filteredAPIRevisions, apiServer)
			}
		}
	} else {
		filteredAPIRevisions = apiServiceInstances
	}

	return filteredAPIRevisions, nil
}

// GetAPIServiceInstances - get management.APIServiceInstance
func (c *ServiceClient) GetAPIServiceInstances(queryParams map[string]string, URL string) ([]*management.APIServiceInstance, error) {
	resources, err := c.GetAPIV1ResourceInstances(queryParams, URL)
	if err != nil {
		return nil, err
	}
	apiServiceIntances, err := management.APIServiceInstanceFromInstanceArray(resources)
	if err != nil {
		return nil, err
	}

	return apiServiceIntances, nil
}

// GetAPIV1ResourceInstances - return apiv1 Resource instance with the default page size
func (c *ServiceClient) GetAPIV1ResourceInstances(queryParams map[string]string, URL string) ([]*apiv1.ResourceInstance, error) {
	return c.GetAPIV1ResourceInstancesWithPageSize(queryParams, URL, apiServerPageSize)
}

// GetAPIV1ResourceInstancesWithPageSize - return apiv1 Resource instance
func (c *ServiceClient) GetAPIV1ResourceInstancesWithPageSize(queryParams map[string]string, URL string, pageSize int) ([]*apiv1.ResourceInstance, error) {
	morePages := true
	page := 1

	resourceInstance := make([]*apiv1.ResourceInstance, 0)

	for morePages {
		query := map[string]string{
			"page":     strconv.Itoa(page),
			"pageSize": strconv.Itoa(pageSize),
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

		if len(resourceInstancePage) < pageSize {
			morePages = false
		} else {
			log.Trace("More resource instance pages exist.  Continue retrieval of resource instances.")
		}

		page++
	}

	return resourceInstance, nil
}
