package apic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

func (c *ServiceClient) buildAPIServiceRevisionSpec(serviceBody *ServiceBody) v1alpha1.ApiServiceRevisionSpec {
	return v1alpha1.ApiServiceRevisionSpec{
		ApiService: serviceBody.serviceContext.serviceName,
		Definition: v1alpha1.ApiServiceRevisionSpecDefinition{
			Type:  c.getRevisionDefinitionType(*serviceBody),
			Value: base64.StdEncoding.EncodeToString(serviceBody.SpecDefinition),
		},
	}
}

func (c *ServiceClient) buildAPIServiceRevisionResource(serviceBody *ServiceBody, revAttributes map[string]string, revisionName string) *v1alpha1.APIServiceRevision {
	return &v1alpha1.APIServiceRevision{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceRevisionGVK(),
			Name:             revisionName,
			Title:            c.updateAPIServiceRevisionTitle(serviceBody),
			Attributes:       c.buildAPIResourceAttributes(serviceBody, revAttributes, false),
			Tags:             c.mapToTagsArray(serviceBody.Tags),
		},
		Spec: c.buildAPIServiceRevisionSpec(serviceBody),
	}
}

func (c *ServiceClient) updateRevisionResource(revision *v1alpha1.APIServiceRevision, serviceBody *ServiceBody) {
	revision.ResourceMeta.Metadata.ResourceVersion = ""
	revision.Title = serviceBody.NameToPush
	revision.ResourceMeta.Attributes = c.buildAPIResourceAttributes(serviceBody, revision.ResourceMeta.Attributes, false)
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

	var revisionName string
	if serviceBody.AltRevisionPrefix == "" {
		revisionPrefix := c.getRevisionPrefix(serviceBody)
		revisionName = revisionPrefix + "." + strconv.Itoa(serviceBody.serviceContext.revisionCount+1)
	} else {
		revisionName = serviceBody.AltRevisionPrefix
	}
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
				return errors.New(err.Error() + rollbackErr.Error())
			}
		}
		return err
	}

	serviceBody.serviceContext.currentRevision = revisionName

	return nil
}

// GetAPIRevisions - Returns the list of API revisions for the specified filter
// NOTE : this function can go away.  You can call GetAPIServiceRevisions directly from your function to get []*v1alpha1.APIServiceRevision
func (c *ServiceClient) GetAPIRevisions(queryParams map[string]string, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	revisions, err := c.GetAPIServiceRevisions(queryParams, c.cfg.GetRevisionsURL(), stage)
	if err != nil {
		return nil, err
	}

	return revisions, nil
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
		queryParams := map[string]string{
			"query": "metadata.references.name==" + serviceBody.serviceContext.serviceName,
			"sort":  "metadata.audit.createTimestamp,DESC",
		}

		revisions, err := c.GetAPIServiceRevisions(queryParams, c.cfg.GetRevisionsURL(), serviceBody.Stage)
		if err != nil {
			return err
		}

		if revisions != nil {
			serviceBody.serviceContext.revisionCount = len(revisions)
			if len(revisions) > 0 {
				serviceBody.serviceContext.previousRevision = revisions[0]
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
	if serviceBody.ResourceType == "" {
		return Unstructured
	}
	return serviceBody.ResourceType
}
