package apic

import (
	"encoding/json"
	"strconv"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/definitions/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func (c *ServiceClient) fubar() error {
	query := map[string]string{
		"query": "metadata.references.name==" + "scrum",
		"sort":  "metadata.audit.createTimestamp,DESC",
	}
	resources, err := c.getAPIResources(query, c.cfg.GetInstancesURL(), "")
	if err != nil {
		return err
	}

	resources = resources.([]v1alpha1.APIServiceInstance)
}

func (c *ServiceClient) getAPIResources(queryParams map[string]string, URL, stage string) ([]interface{}, error) {
	morePages := true
	page := 1

	resourceInstance := make([]interface{}, 0)

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
			log.Debugf("Error while retrieving apirevisions: %s", err.Error())
			return nil, err
		}

		resourceInstancePage := make([]interface{}, 0)
		json.Unmarshal(response, &resourceInstancePage)

		resourceInstance = append(resourceInstance, resourceInstancePage...)

		if len(resourceInstancePage) < apiServerPageSize {
			morePages = false
		}
		page++
	}

	return resourceInstance, nil
}
