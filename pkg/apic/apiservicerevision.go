package apic

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

//TODO
/*
	1. Search for comment "DEPRECATED to be removed on major release"
	2. Remove deprecated code left from APIGOV-19751
*/

const (
	apiSvcRevTemplate = "{{.APIServiceName}} - {{.Date}} - r {{.Revision}}"
	defaultDateFormat = "2006/01/02"
)

// APIServiceRevisionTitle - apiservicerevision template for title
type APIServiceRevisionTitle struct {
	APIServiceName string
	Date           string
	Revision       string
}

// apiSvcRevTitleDateMap - map of date formats for apiservicerevision title
var apiSvcRevTitleDateMap = map[string]string{
	"MM-DD-YYYY": "01-02-2006",
	"MM/DD/YYYY": "01/02/2006",
	"YYYY-MM-DD": "2006-01-02",
	"YYYY/MM/DD": defaultDateFormat,
}

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
		Spec:  c.buildAPIServiceRevisionSpec(serviceBody),
		Owner: c.getOwnerObject(serviceBody, false),
	}
}

func (c *ServiceClient) updateRevisionResource(revision *v1alpha1.APIServiceRevision, serviceBody *ServiceBody) *v1alpha1.APIServiceRevision {
	revision.ResourceMeta.Metadata.ResourceVersion = ""
	revision.Title = serviceBody.NameToPush
	revision.ResourceMeta.Attributes = c.buildAPIResourceAttributes(serviceBody, revision.ResourceMeta.Attributes, false)
	revision.ResourceMeta.Tags = c.mapToTagsArray(serviceBody.Tags)
	revision.Spec = c.buildAPIServiceRevisionSpec(serviceBody)
	revision.Owner = c.getOwnerObject(serviceBody, false)
	return revision
}

//processRevision -
func (c *ServiceClient) processRevision(serviceBody *ServiceBody) error {
	err := c.setRevisionAction(serviceBody)

	if err != nil {
		return err
	}

	var httpMethod string
	revAttributes := serviceBody.RevisionAttributes
	if revAttributes == nil {
		revAttributes = make(map[string]string)
	}
	revisionPrefix := c.getRevisionPrefix(serviceBody)
	revisionName := revisionPrefix + "." + strconv.Itoa(serviceBody.serviceContext.revisionCount)
	revisionURL := c.cfg.GetRevisionsURL()
	revision := serviceBody.serviceContext.previousRevision

	if serviceBody.AltRevisionPrefix != "" {
		revisionName = serviceBody.AltRevisionPrefix
	}

	if serviceBody.serviceContext.revisionAction == updateAPI {
		httpMethod = http.MethodPut
		revisionURL += "/" + revisionName

		revision = c.updateRevisionResource(revision, serviceBody)
		log.Infof("Updating API Service revision for %v-%v in environment %v", serviceBody.APIName, serviceBody.Version, c.cfg.GetEnvironmentName())
	}

	if serviceBody.serviceContext.revisionAction == addAPI {
		httpMethod = http.MethodPost

		// revisionCount is the total number of revisions so far. Add 1 since the action is to create a new revision.
		serviceBody.serviceContext.revisionCount = serviceBody.serviceContext.revisionCount + 1
		revisionCount := serviceBody.serviceContext.revisionCount

		if serviceBody.AltRevisionPrefix == "" {
			revisionName = revisionPrefix + "." + strconv.Itoa(revisionCount)
		}
		if serviceBody.serviceContext.previousRevision != nil {
			revAttributes[AttrPreviousAPIServiceRevisionID] = serviceBody.serviceContext.previousRevision.Metadata.ID
		}
		revision = c.buildAPIServiceRevisionResource(serviceBody, revAttributes, revisionName)
		log.Infof("Creating API Service revision for %v-%v in environment %v", serviceBody.APIName, serviceBody.Version, c.cfg.GetEnvironmentName())
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

	serviceBody.serviceContext.revisionName = revisionName

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

//DEPRECATED to be removed on major release - else function for dateRegEx.MatchString(apiSvcRevPattern) will no longer be needed after "${tag} is invalid"
// updateAPIServiceRevisionTitle - update title after creating or updating APIService Revision according to the APIServiceRevision Pattern
func (c *ServiceClient) updateAPIServiceRevisionTitle(serviceBody *ServiceBody) string {
	apiSvcRevPattern := c.cfg.GetAPIServiceRevisionPattern() // "{{.APIServiceName}} - {{.Date:YYYY/MM/DD}} - r {{.Revision}}"
	dateRegEx := regexp.MustCompile(`{{.Date:.*?}}`)

	var dateFormat = ""

	if dateRegEx.MatchString(apiSvcRevPattern) {
		datePattern := dateRegEx.FindString(apiSvcRevPattern)                              //{{date:YYYY/MM/DD}} or one of the validate formats from apiSvcRevTitleDateMap
		index := strings.Index(datePattern, ":")                                           // get index of ":" (colon)
		date := datePattern[index+1 : index+11]                                            // sub out "{{date:" and "}}" to get the format of the date only
		dateFormat = apiSvcRevTitleDateMap[date]                                           // make sure dateFormat is a valid date format
		apiSvcRevPattern = strings.Replace(apiSvcRevPattern, datePattern, "{{.Date}}", -1) // Once we have the date format, change template to correct {{.Date}} tag
		if dateFormat == "" {
			// Customer is entered an incorrect date format.  Set template and pattern to defaults.
			log.Warnf("CENTRAL_APISERVICEREVISIONPATTERN is returning an invalid {{date:*}} format. Setting format to YYYY-MM-DD")
			apiSvcRevPattern = apiSvcRevTemplate
			dateFormat = defaultDateFormat
		}
	} else {
		// Customer is still using deprecated date format.  Set template and pattern to defaults.
		log.DeprecationWarningDoc("{{date:*}} format for CENTRAL_APISERVICEREVISIONPATTERN", "valid {{.Date:*}} formats")
		apiSvcRevPattern = apiSvcRevTemplate
		dateFormat = defaultDateFormat
	}

	// Build default apiSvcRevTitle.  To be used in case of error processing
	defaultAPISvcRevTitle := fmt.Sprintf(
		"%s - %s - r %s",
		serviceBody.APIName,
		time.Now().Format(dateFormat),
		strconv.Itoa(serviceBody.serviceContext.revisionCount),
	)

	// create apiservicerevision template
	apiSvcRevTitleTemplate := APIServiceRevisionTitle{
		serviceBody.APIName,
		time.Now().Format(dateFormat),
		strconv.Itoa(serviceBody.serviceContext.revisionCount),
	}

	title, err := template.New("apiSvcRevTitle").Parse(apiSvcRevPattern)
	if err != nil {
		log.Warnf("Could not render CENTRAL_APISERVICEREVISIONPATTERN. Returning %s", defaultAPISvcRevTitle)
		return defaultAPISvcRevTitle
	}

	var apiSvcRevTitle bytes.Buffer

	err = title.Execute(&apiSvcRevTitle, apiSvcRevTitleTemplate)
	if err != nil {
		log.Warnf("Could not render CENTRAL_APISERVICEREVISIONPATTERN. Please refer to axway.docs regarding valid CENTRAL_APISERVICEREVISIONPATTERN. Returning %s", defaultAPISvcRevTitle)
		return defaultAPISvcRevTitle
	}

	log.Tracef("Returning apiservicerevision title : %s", apiSvcRevTitle.String())
	return apiSvcRevTitle.String()
}

// GetAPIRevisionByName - Returns the API revision based on its revision name
func (c *ServiceClient) GetAPIRevisionByName(revisionName string) (*v1alpha1.APIServiceRevision, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.cfg.GetRevisionsURL() + "/" + revisionName,
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		if response.Code != http.StatusNotFound {
			responseErr := readResponseErrors(response.Code, response.Body)
			return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
		}
		return nil, nil
	}
	apiRevision := new(v1alpha1.APIServiceRevision)
	json.Unmarshal(response.Body, apiRevision)
	return apiRevision, nil
}
