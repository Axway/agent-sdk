package apic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
func (c *ServiceClient) GetAPIV1ResourceInstances(queryParams map[string]string, url string) ([]*apiv1.ResourceInstance, error) {
	return c.GetAPIV1ResourceInstancesWithPageSize(queryParams, url, c.cfg.GetPageSize())
}

func (c *ServiceClient) getPageSize(url string) (int, bool) {
	c.pageSizeMutex.Lock()
	defer c.pageSizeMutex.Unlock()
	size, ok := c.pageSizes[url]
	return size, ok
}

func (c *ServiceClient) setPageSize(url string, size int) {
	c.pageSizeMutex.Lock()
	defer c.pageSizeMutex.Unlock()
	c.pageSizes[url] = size
}

// GetAPIV1ResourceInstancesWithPageSize - return apiv1 Resource instance
func (c *ServiceClient) GetAPIV1ResourceInstancesWithPageSize(queryParams map[string]string, url string, pageSize int) ([]*apiv1.ResourceInstance, error) {
	initPageSize := pageSize
	morePages := true
	page := 1
	retries := 3

	resourceInstance := make([]*apiv1.ResourceInstance, 0)

	log := c.logger.WithField("endpoint", url)
	log.Trace("retrieving all resources from endpoint")
	if !strings.HasPrefix(url, c.cfg.GetAPIServerURL()) {
		url = c.createAPIServerURL(url)
	}

	// update page size if this endpoint used an adjusted page size before
	if size, ok := c.getPageSize(url); ok {
		pageSize = size
	}

	for morePages {
		query := map[string]string{
			"page":     strconv.Itoa(page),
			"pageSize": strconv.Itoa(pageSize),
		}
		log := log.WithField("page", page).WithField("pageSize", pageSize)

		// Add query params for getting revisions for the service and use the latest one as last reference
		for key, value := range queryParams {
			query[key] = value
		}

		response, err := c.ExecuteAPI(coreapi.GET, url, query, nil)

		if err != nil && retries > 0 && strings.Contains(err.Error(), "context deadline exceeded") {
			// in case of context deadline, lets reduce the page size and restart retrieving the resources
			page = 1
			resourceInstance = make([]*apiv1.ResourceInstance, 0)
			pageSize = pageSize / 2
			log.WithError(err).WithField("newPageSize", pageSize).Debug("error while retrieving resources, retrying with smaller page size")
			retries--

			// update the page size map so this endpoint uses the same size next time
			if pageSize < 1 {
				pageSize = 1
			}
			c.setPageSize(url, pageSize)
			continue
		} else if err != nil {
			log.WithError(err).Debug("error while retrieving resources")
			return nil, err
		}

		resourceInstancePage := make([]*apiv1.ResourceInstance, 0)
		json.Unmarshal(response, &resourceInstancePage)

		resourceInstance = append(resourceInstance, resourceInstancePage...)

		if len(resourceInstancePage) < pageSize || len(resourceInstancePage) == 0 {
			morePages = false
		} else {
			log.Trace("continue retrieving resources from next page")
		}

		page++
	}

	return resourceInstance, nil
}
