package apic

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Axway/agent-sdk/pkg/util"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// TODO
/*
	1. Search for comment "DEPRECATED to be removed on major release"
	2. Remove deprecated code left from APIGOV-19751
*/

const (
	apiSvcRevTemplate = "{{.APIServiceName}}{{if ne .Stage \"\"}} ({{.StageLabel}}: {{.Stage}}){{end}} - {{.Date:YYYY/MM/DD}} - r {{.Revision}}"
	defaultDateFormat = "2006/01/02"
	specHashes        = "specHashes"
)

// APIServiceRevisionTitle - apiservicerevision template for title
type APIServiceRevisionTitle struct {
	APIServiceName string
	Date           string
	Revision       string
	StageLabel     string
	Stage          string
}

// apiSvcRevTitleDateMap - map of date formats for apiservicerevision title
var apiSvcRevTitleDateMap = map[string]string{
	"MM-DD-YYYY": "01-02-2006",
	"MM/DD/YYYY": "01/02/2006",
	"YYYY-MM-DD": "2006-01-02",
	"YYYY/MM/DD": defaultDateFormat,
}

func (c *ServiceClient) buildAPIServiceRevision(serviceBody *ServiceBody) *management.APIServiceRevision {
	newRev := management.NewAPIServiceRevision("", c.cfg.GetEnvironmentName())
	newRev.Title = c.updateAPIServiceRevisionTitle(serviceBody)
	newRev.Attributes = util.CheckEmptyMapStringString(serviceBody.RevisionAttributes)
	newRev.Tags = mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish())
	newRev.Spec = buildAPIServiceRevisionSpec(serviceBody)
	newRev.Owner, _ = c.getOwnerObject(serviceBody, false)

	revDetails := util.MergeMapStringInterface(serviceBody.ServiceAgentDetails, serviceBody.RevisionAgentDetails)
	agentDetails := buildAgentDetailsSubResource(serviceBody, false, revDetails)
	util.SetAgentDetails(newRev, agentDetails)

	return newRev
}

// processRevision -
func (c *ServiceClient) processRevision(serviceBody *ServiceBody) error {
	log := c.logger
	logProcess := "Creating"

	apiServiceRevisions := make([]*management.APIServiceRevision, 0)
	if serviceBody.serviceContext.serviceAction == updateAPI {
		logProcess = "Updating"

		// get current revisions
		revisions, totalCount, err := c.getRevisions("metadata.references.id==" + serviceBody.serviceContext.serviceID)
		if err != nil {
			return err
		}
		serviceBody.serviceContext.revisionCount = totalCount
		apiServiceRevisions = revisions
	}

	// check if a revision with the same hash was already published
	if revName, found := serviceBody.specHashes[serviceBody.specHash]; found {
		name := revName.(string)

		for _, apiServiceRevision := range apiServiceRevisions {
			if apiServiceRevision.Name == name {
				// check to see if the tags have changed from the latest
				if !c.shouldUpdateTags(serviceBody.Tags, apiServiceRevision.Tags) {
					serviceBody.serviceContext.revisionName = name
					return nil
				}
				break // tags have changed. Break and update apiServiceRevision with the latest tags
			}
		}
	}

	log.
		WithField("action", logProcess).
		WithField("service", serviceBody.APIName).
		WithField("version", serviceBody.Version).
		WithField("environment", c.cfg.GetEnvironmentName()).
		Info("process revision")

	rev, err := c.CreateOrUpdateResource(c.buildAPIServiceRevision(serviceBody))
	if err != nil {
		if serviceBody.serviceContext.serviceAction == addAPI {
			_, rollbackErr := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
			if rollbackErr != nil {
				return errors.New(err.Error() + rollbackErr.Error())
			}
		}
		return err
	}

	serviceBody.serviceContext.revisionName = rev.Name

	return nil
}

func (c *ServiceClient) getRevisions(queryString string) ([]*management.APIServiceRevision, int, error) {
	apiServiceRevisions := make([]*management.APIServiceRevision, 0)

	queryParams := map[string]string{
		"query":    queryString,
		"fields":   "name,tags",
		"page":     "1",
		"pageSize": "50",
		"sort":     "metadata.audit.createTimestamp,DESC",
	}
	res, err := c.executeAPI(coreapi.GET, c.cfg.GetRevisionsURL(), queryParams, nil, nil)
	if err != nil {
		return apiServiceRevisions, 0, err
	}
	var count int
	if totalCount, found := res.Headers["X-Axway-Total-Count"]; found {
		count, err = strconv.Atoi(totalCount[0])
		if err != nil {
			return apiServiceRevisions, 0, err
		}
	}

	err = json.Unmarshal(res.Body, &apiServiceRevisions)
	if err != nil {
		return nil, count, err
	}

	return apiServiceRevisions, count, nil
}

// verify last revision tags against the serviceBody tags that are coming in to see if they are equal or not
func (c *ServiceClient) shouldUpdateTags(serviceBodyTags map[string]interface{}, revisionTags []string) bool {
	// Extract values from map and convert to []string
	var mapValues []string
	for _, v := range serviceBodyTags {
		if strVal, ok := v.(string); ok {
			mapValues = append(mapValues, strVal)
		}
	}

	// Sort both slices to allow unordered comparison
	sort.Strings(mapValues)
	sort.Strings(revisionTags)

	// Compare
	equal := reflect.DeepEqual(mapValues, revisionTags)
	return !equal
}

// GetAPIRevisions - Returns the list of API revisions for the specified filter
// NOTE : this function can go away.  You can call GetAPIServiceRevisions directly from your function to get []*management.APIServiceRevision
func (c *ServiceClient) GetAPIRevisions(query map[string]string, stage string) ([]*management.APIServiceRevision, error) {
	revisions, err := c.GetAPIServiceRevisions(query, c.cfg.GetRevisionsURL(), stage)
	if err != nil {
		return nil, err
	}

	return revisions, nil
}

// DEPRECATED to be removed on major release - else function for dateRegEx.MatchString(apiSvcRevPattern) will no longer be needed after "${tag} is invalid"
// updateAPIServiceRevisionTitle - update title after creating or updating APIService Revision according to the APIServiceRevision Pattern
func (c *ServiceClient) updateAPIServiceRevisionTitle(serviceBody *ServiceBody) string {
	apiSvcRevPattern := c.cfg.GetAPIServiceRevisionPattern()
	if apiSvcRevPattern == "" {
		apiSvcRevPattern = apiSvcRevTemplate
	}
	dateRegEx := regexp.MustCompile(`\{\{.Date:.*?\}\}`)

	var dateFormat = ""

	if dateRegEx.MatchString(apiSvcRevPattern) {
		datePattern := dateRegEx.FindString(apiSvcRevPattern)                              // {{.Date:YYYY/MM/DD}} or one of the validate formats from apiSvcRevTitleDateMap
		index := strings.Index(datePattern, ":")                                           // get index of ":" (colon)
		date := datePattern[index+1 : index+11]                                            // sub out "{{.Date:" and "}}" to get the format of the date only
		dateFormat = apiSvcRevTitleDateMap[date]                                           // make sure dateFormat is a valid date format
		apiSvcRevPattern = strings.Replace(apiSvcRevPattern, datePattern, "{{.Date}}", -1) // Once we have the date format, set to {{.Date}} only
		if dateFormat == "" {
			// Customer is entered an incorrect date format.  Set template and pattern to defaults.
			log.Warnf("CENTRAL_APISERVICEREVISIONPATTERN is returning an invalid {{.Date:*}} format. Setting format to YYYY-MM-DD")
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

	stage := serviceBody.Stage
	if serviceBody.StageDisplayName != "" {
		stage = serviceBody.StageDisplayName
	}

	// create apiservicerevision template
	apiSvcRevTitleTemplate := APIServiceRevisionTitle{
		APIServiceName: serviceBody.APIName,
		Date:           time.Now().Format(dateFormat),
		Revision:       strconv.Itoa(serviceBody.serviceContext.revisionCount),
		StageLabel:     serviceBody.StageDescriptor,
		Stage:          stage,
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
func (c *ServiceClient) GetAPIRevisionByName(name string) (*management.APIServiceRevision, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.cfg.GetRevisionsURL() + "/" + name,
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
	apiRevision := new(management.APIServiceRevision)
	err = json.Unmarshal(response.Body, apiRevision)
	return apiRevision, err
}

func buildAPIServiceRevisionSpec(serviceBody *ServiceBody) management.ApiServiceRevisionSpec {
	return management.ApiServiceRevisionSpec{
		ApiService: serviceBody.serviceContext.serviceName,
		Definition: management.ApiServiceRevisionSpecDefinition{
			Type:        getRevisionDefinitionType(*serviceBody),
			Value:       base64.StdEncoding.EncodeToString(serviceBody.SpecDefinition),
			ContentType: serviceBody.ResourceContentType,
			Version:     serviceBody.GetSpecVersion(),
		},
	}
}

func getRevisionPrefix(serviceBody *ServiceBody) string {
	if serviceBody.Stage != "" {
		return sanitizeAPIName(fmt.Sprintf("%s-%s", serviceBody.serviceContext.serviceName, serviceBody.Stage))
	}
	return sanitizeAPIName(serviceBody.serviceContext.serviceName)
}

// getRevisionDefinitionType -
func getRevisionDefinitionType(serviceBody ServiceBody) string {
	if serviceBody.ResourceType == "" {
		return Unstructured
	}
	return serviceBody.ResourceType
}
