package apic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	v1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/tidwall/gjson"
)

func (c *ServiceClient) buildAPIServiceRevisionSpec(serviceBody *ServiceBody) v1alpha1.ApiServiceRevisionSpec {
	return v1alpha1.ApiServiceRevisionSpec{
		ApiService: serviceBody.serviceContext.serviceName,
		Definition: v1alpha1.ApiServiceRevisionSpecDefinition{
			Type:  c.getRevisionDefinitionType(*serviceBody),
			Value: base64.StdEncoding.EncodeToString(serviceBody.Swagger),
		},
	}
}

func (c *ServiceClient) buildAPIServiceRevisionResource(serviceBody *ServiceBody, revAttributes map[string]string, revisionName string) *v1alpha1.APIServiceRevision {
	return &v1alpha1.APIServiceRevision{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceRevisionGVK(),
			Name:             revisionName,
			Title:            serviceBody.NameToPush,
			Attributes:       c.buildAPIResourceAttributes(serviceBody, revAttributes, false),
			Tags:             c.mapToTagsArray(serviceBody.Tags),
		},
		Spec: c.buildAPIServiceRevisionSpec(serviceBody),
	}
}

func (c *ServiceClient) updateRevisionResource(revision *v1alpha1.APIServiceRevision, serviceBody *ServiceBody) {
	revision.Title = serviceBody.NameToPush
	revision.ResourceMeta.Attributes = c.buildAPIResourceAttributes(serviceBody, revision.ResourceMeta.Attributes, true)
	revision.ResourceMeta.Tags = c.mapToTagsArray(serviceBody.Tags)
	revision.Spec = c.buildAPIServiceRevisionSpec(serviceBody)
}

//processRevision -
func (c *ServiceClient) processRevision(serviceBody *ServiceBody) error {
	err := c.setRevisionAction(serviceBody)
	if err != nil {
		return err
	}

	httpMethod := http.MethodPost
	revisionURL := c.cfg.GetRevisionsURL()
	var revAttributes map[string]string

	revisionPrefix := c.getRevisionPrefix(serviceBody)
	revisionName := revisionPrefix + "." + strconv.Itoa(serviceBody.serviceContext.revisionCount+1)
	revision := serviceBody.serviceContext.previousRevision

	if serviceBody.serviceContext.revisionAction == updateAPI {
		revisionName = serviceBody.serviceContext.previousRevision.Name
		httpMethod = http.MethodPut
		revisionURL += "/" + revisionName
		c.updateRevisionResource(revision, serviceBody)
	} else {
		revAttributes = make(map[string]string)
		if serviceBody.serviceContext.previousRevision != nil {
			revAttributes[AttrPreviousAPIServiceRevisionID] = serviceBody.serviceContext.previousRevision.Metadata.ID
		}
		revision = c.buildAPIServiceRevisionResource(serviceBody, revAttributes, revisionName)
	}

	buffer, err := json.Marshal(revision)
	if err != nil {
		return err
	}

	_, err = c.apiServiceDeployAPI(httpMethod, revisionURL, buffer)
	if err != nil {
		if serviceBody.serviceContext.serviceAction == addAPI {
			_, rollbackErr := c.rollbackAPIService(*serviceBody, serviceBody.serviceContext.serviceName)
			if rollbackErr != nil {
				err = rollbackErr
			}
			return err
		}
	} else {
		serviceBody.serviceContext.currentRevision = revisionName
	}

	return nil
}

// getAPIRevisions - Returns the list of API revisions for the specified filter
func (c *ServiceClient) getAPIRevisions(queryParams map[string]string, stage string) ([]v1alpha1.APIServiceRevision, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetRevisionsURL(),
		Headers:     headers,
		QueryParams: queryParams,
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
	revisions := make([]v1alpha1.APIServiceRevision, 0)
	json.Unmarshal(response.Body, &revisions)

	apiServerRevisions := make([]v1alpha1.APIServiceRevision, 0)

	//create array and filter by stage name. Check the stage name as this does not apply for v7
	if stage != "" {
		for _, apiServer := range revisions {
			if strings.Contains(strings.ToLower(apiServer.Name), strings.ToLower(stage)) {
				apiServerRevisions = append(apiServerRevisions, apiServer)
			}
		}
	} else {
		apiServerRevisions = revisions
	}

	return apiServerRevisions, nil

}

func (c *ServiceClient) getRevisionPrefix(serviceBody *ServiceBody) string {
	if serviceBody.Stage != "" {
		return sanitizeAPIName(fmt.Sprintf("%s-%s", serviceBody.serviceContext.serviceName, serviceBody.Stage))
	}
	return sanitizeAPIName(serviceBody.serviceContext.serviceName)
}

func (c *ServiceClient) setRevisionAction(serviceBody *ServiceBody) error {
	// If service is created in the chain, then set action to create revision
	serviceBody.serviceContext.revisionAction = addAPI
	// If service is updated, identify the action based on the existing revisions and update type(minor/major)
	if serviceBody.serviceContext.serviceAction == updateAPI {
		// Get revisions for the service and use the latest one as last reference
		revisionFilter := map[string]string{
			"query": "metadata.references.name==" + serviceBody.serviceContext.serviceName,
			"sort":  "metadata.audit.createTimestamp,DESC",
		}
		revisions, err := c.getAPIRevisions(revisionFilter, serviceBody.Stage)
		if err != nil {
			return err
		}
		if revisions != nil {
			serviceBody.serviceContext.revisionCount = len(revisions)
			if len(revisions) > 0 {
				serviceBody.serviceContext.previousRevision = &revisions[0]
				if serviceBody.APIUpdateSeverity == MinorChange {
					// For minor change use the latest revision and update existing
					serviceBody.serviceContext.revisionAction = updateAPI
				}
			}
		}
	}
	return nil
}

//getRevisionDefinitionType -
func (c *ServiceClient) getRevisionDefinitionType(serviceBody ServiceBody) string {
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
	return revisionDefinitionType
}
