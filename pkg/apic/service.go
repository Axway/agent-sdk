package apic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
)

type actionType int

const (
	addAPI    = iota
	updateAPI = iota
)

const (
	tenMB = 10485760
)

// PublishService - processes the API to create/update apiservice, revision, instance
func (c *ServiceClient) PublishService(serviceBody *ServiceBody) (*management.APIService, error) {
	logger := c.logger.WithField("serviceName", serviceBody.NameToPush).WithField("apiID", serviceBody.RestAPIID)
	if serviceBody.PrimaryKey != "" {
		logger = logger.WithField("primaryKey", serviceBody.PrimaryKey)
	}

	// if the team is set in the config, use that team name and id for all services
	if c.cfg.GetTeamName() != "" {
		if teamID, found := c.getTeamFromCache(c.cfg.GetTeamName()); found {
			serviceBody.TeamName = c.cfg.GetTeamName()
			serviceBody.teamID = teamID
			logger.Debugf("setting team name (%s) and team id (%s)", serviceBody.TeamName, serviceBody.teamID)
		}
	}

	// there is a current envoy restriction with the payload size (10mb). Quick check on the size
	if len(serviceBody.SpecDefinition) >= tenMB {
		// if greater than 10mb, return
		err := fmt.Errorf("service %s carries a payload greater than 10mb. Service not created", serviceBody.APIName)
		logger.WithError(err).Error("error processing service")
		return nil, err
	}

	// API Service
	logger.Trace("processing service")
	apiSvc, err := c.processService(serviceBody)
	if err != nil {
		logger.WithError(err).Error("processing service")
		return nil, err
	}
	// Update description title after creating APIService to include the stage name if it exists
	c.postAPIServiceUpdate(serviceBody)

	// RevisionProcessor
	logger.Trace("processing revision")
	err = c.processRevision(serviceBody)
	if err != nil {
		logger.WithError(err).Error("processing revision")
		return nil, err
	}

	// InstanceProcessor
	logger.Trace("processing instance")
	err = c.processInstance(serviceBody)
	if err != nil {
		logger.WithError(err).Error("processing instance")
		return nil, err
	}

	logger.Trace("adding spec hashes to service")
	serviceBody.specHashes[serviceBody.specHash] = serviceBody.serviceContext.revisionName
	details := util.GetAgentDetails(apiSvc)
	details[specHashes] = serviceBody.specHashes
	util.SetAgentDetails(apiSvc, details)
	if err := c.CreateSubResource(apiSvc.ResourceMeta, map[string]interface{}{defs.XAgentDetails: details}); err != nil {
		logger.Error("error adding spec hashes in x-agent-details, retrying")
		// if the update failed try once more
		if err := c.CreateSubResource(apiSvc.ResourceMeta, map[string]interface{}{defs.XAgentDetails: details}); err != nil {
			logger.WithError(err).Error("could not add spec hashes in x-agent-details")
		}
	}
	ri, _ := apiSvc.AsInstance()
	err = c.caches.AddAPIService(ri)
	if err != nil {
		logger.WithError(err).Error("adding service to cache")
	}

	return apiSvc, nil
}

// DeleteServiceByName -
func (c *ServiceClient) DeleteServiceByName(name string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetServicesURL()+"/"+name, nil)
	return err
}

// DeleteAPIServiceInstance deletes an api service instance in central by name
func (c *ServiceClient) DeleteAPIServiceInstance(name string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetInstancesURL()+"/"+name, nil)
	if err != nil && err.Error() != strconv.Itoa(http.StatusNotFound) {
		return err
	}
	return nil
}

// postApiServiceUpdate - called after APIService was created or updated.
// Update description and title after updating or creating APIService to include the stage name if it exists
func (c *ServiceClient) postAPIServiceUpdate(serviceBody *ServiceBody) {
	if serviceBody.Stage != "" {
		stageDisplay := serviceBody.Stage
		if serviceBody.StageDisplayName != "" {
			stageDisplay = serviceBody.StageDisplayName
		}

		stageDescription := fmt.Sprintf("%s: %s", serviceBody.StageDescriptor, stageDisplay)
		if len(serviceBody.Description) > 0 {
			stageDescription = fmt.Sprintf(", %s", stageDescription)
			if len(serviceBody.Description)+len(stageDescription) >= maxDescriptionLength {
				description := serviceBody.Description[0 : maxDescriptionLength-len(strEllipsis)-len(stageDescription)]
				serviceBody.Description = fmt.Sprintf("%s%s%s", description, strEllipsis, stageDescription)
			} else {
				serviceBody.Description = fmt.Sprintf("%s%s", serviceBody.Description, stageDescription)
			}
		} else {
			serviceBody.Description = stageDescription
		}
		serviceBody.NameToPush = fmt.Sprintf("%v (%s: %v)", serviceBody.NameToPush, serviceBody.StageDescriptor, stageDisplay)
	} else if c.cfg.GetAppendEnvironmentToTitle() {
		// Append the environment name to the title, if set
		serviceBody.NameToPush = fmt.Sprintf("%v (%v)", serviceBody.NameToPush, c.cfg.GetEnvironmentName())
	}
}

func buildAgentDetailsSubResource(
	serviceBody *ServiceBody, isAPIService bool, additional map[string]interface{},
) map[string]interface{} {
	details := make(map[string]interface{})

	externalAPIID := serviceBody.RestAPIID
	// check to see if is an APIService
	if !isAPIService && serviceBody.Stage != "" {
		details[defs.AttrExternalAPIStage] = serviceBody.Stage
	}
	if serviceBody.PrimaryKey != "" {
		details[defs.AttrExternalAPIPrimaryKey] = serviceBody.PrimaryKey
	}

	details[defs.AttrExternalAPIID] = externalAPIID
	details[defs.AttrExternalAPIName] = serviceBody.APIName
	details[defs.AttrCreatedBy] = serviceBody.CreatedBy

	return util.MergeMapStringInterface(details, additional)
}

func isValidAuthPolicy(auth string) bool {
	for _, item := range ValidPolicies {
		if item == auth {
			return true
		}
	}
	return false
}

// Sanitize name to be path friendly and follow this regex: ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*
func sanitizeAPIName(name string) string {
	return util.ConvertToDomainNameCompliant(name)
}

// apiServiceDeployAPI -
func (c *ServiceClient) apiServiceDeployAPI(method, url string, buffer []byte) (*v1.ResourceInstance, error) {
	ri, err := c.executeAPIServiceAPI(method, url, buffer)
	return ri, err
}

// executeAPIServiceAPI -
func (c *ServiceClient) executeAPIServiceAPI(method, url string, buffer []byte) (*v1.ResourceInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
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
		return nil, err
	}
	//  Check to see if rollback was processed
	if method == http.MethodDelete && response.Code == http.StatusNoContent {
		return nil, nil
	}

	if response.Code >= http.StatusBadRequest {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}
	ri := &v1.ResourceInstance{}
	json.Unmarshal(response.Body, ri)
	return ri, nil
}
