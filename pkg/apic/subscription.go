package apic

import (
	"encoding/json"
	"fmt"
	"net/http"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	uc "github.com/Axway/agent-sdk/pkg/apic/unifiedcatalog/models"
	"github.com/Axway/agent-sdk/pkg/util"
	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"
	log "github.com/Axway/agent-sdk/pkg/util/log"
)

// SubscriptionState - Type definition for subscription state
type SubscriptionState string

// SubscriptionState
const (
	SubscriptionApproved              = SubscriptionState("APPROVED")
	SubscriptionRequested             = SubscriptionState("REQUESTED")
	SubscriptionRejected              = SubscriptionState("REJECTED")
	SubscriptionActive                = SubscriptionState("ACTIVE")
	SubscriptionUnsubscribed          = SubscriptionState("UNSUBSCRIBED")
	SubscriptionUnsubscribeInitiated  = SubscriptionState("UNSUBSCRIBE_INITIATED")
	SubscriptionFailedToSubscribe     = SubscriptionState("FAILED_TO_SUBSCRIBE")
	SubscriptionFailedToUnsubscribe   = SubscriptionState("FAILED_TO_UNSUBSCRIBE")
	AccessRequestProvisioning         = SubscriptionState("provisioning")
	AccessRequestProvisioned          = SubscriptionState("provisioned")
	AccessRequestFailedProvisioning   = SubscriptionState("failedProvisioning")
	AccessRequestDeprovisioning       = SubscriptionState("deprovisioning")
	AccessRequestDeprovisioned        = SubscriptionState("deprovisioned")
	AccessRequestFailedDeprovisioning = SubscriptionState("failedDeprovisioning")
)

var catalogToAccessRequestStateMap = map[SubscriptionState]SubscriptionState{
	SubscriptionApproved:             AccessRequestProvisioning,
	SubscriptionRequested:            AccessRequestFailedProvisioning,
	SubscriptionRejected:             AccessRequestFailedProvisioning,
	SubscriptionActive:               AccessRequestProvisioned,
	SubscriptionUnsubscribed:         AccessRequestDeprovisioned,
	SubscriptionUnsubscribeInitiated: AccessRequestDeprovisioning,
	SubscriptionFailedToSubscribe:    AccessRequestFailedProvisioning,
	SubscriptionFailedToUnsubscribe:  AccessRequestFailedDeprovisioning,
}

// getAccessRequestState - gets the access request state equivalent from a subscription state
func (s SubscriptionState) getAccessRequestState() SubscriptionState {
	if state, found := catalogToAccessRequestStateMap[s]; found {
		return state
	}
	return s
}

// isUnifiedCatalogState - returns true is the state is a unified catalog state
func (s SubscriptionState) isUnifiedCatalogState() bool {
	_, found := catalogToAccessRequestStateMap[s]
	return found
}

const (
	appNameKey              = "appName"
	subscriptionAppNameType = "string"
	profileKey              = "profile"
)

// Subscription -
type Subscription interface {
	GetID() string
	GetName() string
	GetApicID() string
	GetRemoteAPIAttributes() map[string]string
	GetRemoteAPIID() string
	GetRemoteAPIStage() string
	GetCatalogItemID() string
	GetCreatedUserID() string
	GetState() SubscriptionState
	GetPropertyValue(propertyKey string) string
	UpdateState(newState SubscriptionState, description string) error
	UpdateStateWithProperties(newState SubscriptionState, description string, properties map[string]interface{}) error
	UpdateEnumProperty(key, value, dataType string) error
	UpdateProperties(appName string) error
	UpdatePropertyValues(values map[string]interface{}) error
	setAPIResourceInfo(apiServerResource *v1.ResourceInstance)
}

// CentralSubscription -
type CentralSubscription struct {
	Subscription
	CatalogItemSubscription *uc.CatalogItemSubscription `json:"catalogItemSubscription"`
	ApicID                  string                      `json:"-"`
	RemoteAPIID             string                      `json:"-"`
	RemoteAPIStage          string                      `json:"-"`
	apicClient              *ServiceClient
	RemoteAPIAttributes     map[string]string
}

// GetRemoteAPIAttributes - Returns the attributes from the API that the subscription is tied to.
func (s *CentralSubscription) GetRemoteAPIAttributes() map[string]string {
	return s.RemoteAPIAttributes
}

// GetCreatedUserID - Returns ID of the user that created the subscription
func (s *CentralSubscription) GetCreatedUserID() string {
	return s.CatalogItemSubscription.Metadata.CreateUserId
}

// GetID - Returns ID of the subscription
func (s *CentralSubscription) GetID() string {
	return s.CatalogItemSubscription.Id
}

// GetName - Returns Name of the subscription
func (s *CentralSubscription) GetName() string {
	return s.CatalogItemSubscription.Name
}

// GetApicID - Returns ID of the Catalog Item or API Service instance
func (s *CentralSubscription) GetApicID() string {
	return s.ApicID
}

// GetRemoteAPIID - Returns ID of the API on remote gateway
func (s *CentralSubscription) GetRemoteAPIID() string {
	return s.RemoteAPIID
}

// GetRemoteAPIStage - Returns the stage name of the API on remote gateway
func (s *CentralSubscription) GetRemoteAPIStage() string {
	return s.RemoteAPIStage
}

// GetCatalogItemID - Returns ID of the Catalog Item
func (s *CentralSubscription) GetCatalogItemID() string {
	return s.CatalogItemSubscription.CatalogItemId
}

// GetState - Returns subscription state
func (s *CentralSubscription) GetState() SubscriptionState {
	return SubscriptionState(s.CatalogItemSubscription.State)
}

// GetPropertyValue - Returns subscription Property value based on the key
func (s *CentralSubscription) GetPropertyValue(propertyKey string) string {
	if len(s.CatalogItemSubscription.Properties) > 0 {
		subscriptionProperty := s.CatalogItemSubscription.Properties[0]
		value, ok := subscriptionProperty.Value[propertyKey]
		if ok {
			return fmt.Sprintf("%v", value)
		}
	}
	return ""
}

func (s *CentralSubscription) updateProperties(properties map[string]interface{}) error {
	if len(properties) == 0 {
		return nil
	}

	// keep existing properties
	var profile map[string]interface{}
	for _, p := range s.CatalogItemSubscription.Properties {
		if p.Key == profileKey {
			profile = p.Value
		}
	}

	allProps := map[string]interface{}{}
	// keep existing properties
	for k, v := range profile {
		allProps[k] = v
	}

	// override with new values
	for k, v := range properties {
		allProps[k] = v
	}

	return s.updatePropertyValue(profileKey, allProps)
}

func (s *CentralSubscription) updateCatalogSubscriptionState(newState SubscriptionState, description string) (*coreapi.Request, error) {
	headers, err := s.getServiceClient().createHeader()
	if err != nil {
		return nil, err
	}

	// Catalog has a requirement for the description to be < 350 characters.
	if len(description) > 350 {
		log.Warnf("Truncating description. Description to update catalog subscription state is greater than the 350 allowable characters [%s]", description)
		description = description[:350]
	}

	subState := uc.CatalogItemSubscriptionState{
		Description: description,
		State:       string(newState),
	}
	statePostBody, err := json.Marshal(subState)
	if err != nil {
		return nil, err
	}
	return &coreapi.Request{
		Method:      coreapi.POST,
		URL:         s.getServiceClient().cfg.GetCatalogItemSubscriptionStatesURL(s.GetCatalogItemID(), s.GetID()),
		QueryParams: nil,
		Headers:     headers,
		Body:        statePostBody,
	}, nil
}

// UpdateStateWithProperties - Updates the state of subscription
func (s *CentralSubscription) UpdateStateWithProperties(newState SubscriptionState, description string, properties map[string]interface{}) error {
	if err := s.updateProperties(properties); err != nil {
		return err
	}

	request, err := s.updateCatalogSubscriptionState(newState, description)
	if err != nil {
		return err
	}

	if response, err := s.getServiceClient().apiClient.Send(*request); err != nil {
		return agenterrors.Wrap(ErrSubscriptionQuery, err.Error())
	} else if !(response.Code == http.StatusOK || response.Code == http.StatusCreated) {
		readResponseErrors(response.Code, response.Body)
		return ErrSubscriptionResp.FormatError(response.Code)
	}
	return nil
}

// UpdateState - Updates the state of subscription
func (s *CentralSubscription) UpdateState(newState SubscriptionState, description string) error {
	return s.UpdateStateWithProperties(newState, description, map[string]interface{}{})
}

// getServiceClient - returns the apic client
func (s *CentralSubscription) getServiceClient() *ServiceClient {
	return s.apicClient
}

// UpdateEnumProperty -
func (s *CentralSubscription) UpdateEnumProperty(key, newValue, dataType string) error {
	catalogItemID := s.GetCatalogItemID()

	// First need to get the subscriptionDefProperties for the catalog item
	ss, err := s.getServiceClient().GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, profileKey)
	if ss == nil || err != nil {
		return agenterrors.Wrap(ErrGetSubscriptionDefProperties, err.Error())
	}

	// update the appName in the enum
	prop := ss.GetProperty(key)

	// first check that the property is unique
	for _, ele := range prop.Enum {
		if ele == newValue {
			return nil
		}
	}
	newOptions := append(prop.Enum, newValue)

	ss.AddProperty(key, dataType, prop.Description, "", true, newOptions)
	// note: there will be a small time window where the enum items might be out-of-order. The agent will eventually
	// pick up the changes and update the schema, which will reorder them.

	// update the the subscriptionDefProperties for the catalog item. This MUST be done before updating the subscription
	err = s.getServiceClient().UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, profileKey, ss)
	if err != nil {
		return agenterrors.Wrap(ErrUpdateSubscriptionDefProperties, err.Error())
	}

	return nil
}

// UpdateProperties -
func (s *CentralSubscription) UpdateProperties(appName string) error {
	err := s.UpdateEnumProperty(appNameKey, appName, subscriptionAppNameType)
	if err != nil {
		return err
	}

	// Now we can update the appname in the subscription itself
	err = s.updatePropertyValue(profileKey, map[string]interface{}{appNameKey: appName})
	if err != nil {
		return agenterrors.Wrap(ErrUpdateSubscriptionDefProperties, err.Error())
	}

	return nil
}

// updatePropertyValue - Updates the property value of the subscription
func (s *CentralSubscription) updatePropertyValue(propertyKey string, value map[string]interface{}) error {
	headers, err := s.getServiceClient().createHeader()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", s.getServiceClient().cfg.GetCatalogItemSubscriptionPropertiesURL(s.GetCatalogItemID(), s.GetID()), propertyKey)

	body, err := json.Marshal(value)
	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.PUT,
		URL:     url,
		Headers: headers,
		Body:    body,
	}

	response, err := s.getServiceClient().apiClient.Send(request)
	if err != nil {
		return err
	}

	if !(response.Code == http.StatusOK) {
		readResponseErrors(response.Code, response.Body)
		return ErrSubscriptionResp.FormatError(response.Code)
	}
	return nil
}

// UpdatePropertyValues - Updates the property values of the subscription
func (s *CentralSubscription) UpdatePropertyValues(values map[string]interface{}) error {
	headers, err := s.getServiceClient().createHeader()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", s.getServiceClient().cfg.GetCatalogItemSubscriptionPropertiesURL(s.GetCatalogItemID(), s.GetID()), profileKey)
	body, err := json.Marshal(values)

	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.PUT,
		URL:     url,
		Headers: headers,
		Body:    body,
	}

	response, err := s.getServiceClient().apiClient.Send(request)
	if err != nil {
		return err
	}

	if !(response.Code == http.StatusOK) {
		readResponseErrors(response.Code, response.Body)
		return ErrSubscriptionResp.FormatError(response.Code)
	}
	return nil
}

func (s *CentralSubscription) setAPIResourceInfo(apiServerResource *v1.ResourceInstance) {
	s.ApicID = apiServerResource.Metadata.ID
	apiID, _ := util.GetAgentDetailsValue(apiServerResource, defs.AttrExternalAPIID)
	stage, _ := util.GetAgentDetailsValue(apiServerResource, defs.AttrExternalAPIStage)
	s.RemoteAPIID = apiID
	s.RemoteAPIStage = stage

	// get the x agent details for this, convert to map[string]string
	s.RemoteAPIAttributes = util.MapStringInterfaceToStringString(util.GetAgentDetails(apiServerResource))
}

// AccessRequestSubscription -
type AccessRequestSubscription struct {
	Subscription
	AccessRequest       *v1alpha1.AccessRequest `json:"accessRequest"`
	ApicID              string                  `json:"-"`
	RemoteAPIID         string                  `json:"-"`
	RemoteAPIStage      string                  `json:"-"`
	apicClient          *ServiceClient
	RemoteAPIAttributes map[string]string
}

// GetRemoteAPIAttributes - Returns the attributes from the API that the subscription is tied to.
func (s *AccessRequestSubscription) GetRemoteAPIAttributes() map[string]string {
	return s.RemoteAPIAttributes
}

// GetCreatedUserID - Returns ID of the user that created the subscription
func (s *AccessRequestSubscription) GetCreatedUserID() string {
	return s.AccessRequest.Metadata.Audit.CreateUserID
}

// GetID - Returns Name of the subscription
func (s *AccessRequestSubscription) GetID() string {
	return s.AccessRequest.Name
}

// GetName - Returns Name of the subscription
func (s *AccessRequestSubscription) GetName() string {
	return s.AccessRequest.Name
}

// GetApicID - Returns ID of the API Service instance
func (s *AccessRequestSubscription) GetApicID() string {
	return s.AccessRequest.Spec.ApiServiceInstance
}

// GetRemoteAPIID - Returns ID of the API on remote gateway
func (s *AccessRequestSubscription) GetRemoteAPIID() string {
	return s.RemoteAPIID
}

// GetRemoteAPIStage - Returns the stage name of the API on remote gateway
func (s *AccessRequestSubscription) GetRemoteAPIStage() string {
	return s.RemoteAPIStage
}

// GetCatalogItemID - Returns the name of the accesd request
func (s *AccessRequestSubscription) GetCatalogItemID() string {
	return s.AccessRequest.Name
}

// GetState - Returns subscription state
func (s *AccessRequestSubscription) GetState() SubscriptionState {
	return SubscriptionState(s.AccessRequest.State.Name)
}

// GetPropertyValue - Returns subscription Property value based on the key
func (s *AccessRequestSubscription) GetPropertyValue(propertyKey string) string {
	if value, found := s.AccessRequest.Spec.Data[propertyKey]; found {
		return value.(string)
	}
	return ""
}

func (s *AccessRequestSubscription) updateProperties(properties map[string]interface{}) error {
	if len(properties) == 0 {
		return nil
	}

	// override with new values
	for k, v := range properties {
		s.AccessRequest.Spec.Data[k] = v
	}

	return nil
}

func (s *AccessRequestSubscription) updateAccessRequestState(newState SubscriptionState, description string) (*coreapi.Request, *coreapi.Request, error) {
	headers, err := s.getServiceClient().createHeader()
	if err != nil {
		return nil, nil, err
	}

	s.AccessRequest.State = v1alpha1.AccessRequestState{
		Message: description,
		Name:    string(newState.getAccessRequestState()),
	}
	statePostBody, err := json.Marshal(s.AccessRequest)
	if err != nil {
		return nil, nil, err
	}

	return &coreapi.Request{
			Method:      coreapi.PUT,
			URL:         s.getServiceClient().cfg.GetAccessRequestURL(s.GetName()),
			QueryParams: nil,
			Headers:     headers,
			Body:        statePostBody,
		},
		&coreapi.Request{
			Method:      coreapi.PUT,
			URL:         s.getServiceClient().cfg.GetAccessRequestStateURL(s.GetName()),
			QueryParams: nil,
			Headers:     headers,
			Body:        statePostBody,
		},
		nil
}

// UpdateStateWithProperties - Updates the state of subscription
func (s *AccessRequestSubscription) UpdateStateWithProperties(newState SubscriptionState, description string, properties map[string]interface{}) error {
	if err := s.updateProperties(properties); err != nil {
		return err
	}

	propsRequest, stateRequest, err := s.updateAccessRequestState(newState, description)
	if err != nil {
		return err
	}

	for _, request := range []*coreapi.Request{stateRequest, propsRequest} {
		if response, err := s.getServiceClient().apiClient.Send(*request); err != nil {
			return agenterrors.Wrap(ErrSubscriptionQuery, err.Error())
		} else if !(response.Code == http.StatusOK || response.Code == http.StatusCreated) {
			readResponseErrors(response.Code, response.Body)
			return ErrSubscriptionResp.FormatError(response.Code)
		}
	}
	return nil
}

// UpdateState - Updates the state of subscription
func (s *AccessRequestSubscription) UpdateState(newState SubscriptionState, description string) error {
	return s.UpdateStateWithProperties(newState, description, map[string]interface{}{})
}

// getServiceClient - returns the apic client
func (s *AccessRequestSubscription) getServiceClient() *ServiceClient {
	return s.apicClient
}

// UpdateEnumProperty - not used on access request
func (s *AccessRequestSubscription) UpdateEnumProperty(key, newValue, dataType string) error {
	return nil
}

// UpdateProperties - not used on access request
func (s *AccessRequestSubscription) UpdateProperties(appName string) error {
	return nil
}

// UpdatePropertyValues - Updates the property values of the subscription
func (s *AccessRequestSubscription) UpdatePropertyValues(values map[string]interface{}) error {
	headers, err := s.getServiceClient().createHeader()
	if err != nil {
		return err
	}

	if s.AccessRequest.Spec.Data == nil {
		s.AccessRequest.Spec.Data = map[string]interface{}{}
	}
	for key, val := range values {
		s.AccessRequest.Spec.Data[key] = val
	}

	url := s.getServiceClient().cfg.GetAccessRequestURL(s.GetName())
	body, err := json.Marshal(s.AccessRequest)

	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.PUT,
		URL:     url,
		Headers: headers,
		Body:    body,
	}

	response, err := s.getServiceClient().apiClient.Send(request)
	if err != nil {
		return err
	}

	if response.Code != http.StatusOK {
		readResponseErrors(response.Code, response.Body)
		return ErrSubscriptionResp.FormatError(response.Code)
	}
	return nil
}

func (s *AccessRequestSubscription) setAPIResourceInfo(apiServerResource *v1.ResourceInstance) {
	s.ApicID = apiServerResource.Metadata.ID
	apiID, _ := util.GetAgentDetailsValue(apiServerResource, defs.AttrExternalAPIID)
	stage, _ := util.GetAgentDetailsValue(apiServerResource, defs.AttrExternalAPIStage)
	s.RemoteAPIID = apiID
	s.RemoteAPIStage = stage
	s.RemoteAPIAttributes = apiServerResource.Attributes
}

// getSubscriptions -
func (c *ServiceClient) getSubscriptions(states []string) ([]CentralSubscription, error) {
	queryParams := make(map[string]string)

	searchQuery := ""
	for _, state := range states {
		if searchQuery != "" {
			searchQuery += ","
		}
		searchQuery += "state==" + state
	}

	queryParams["query"] = searchQuery

	resBody, err := c.sendSubscriptionsRequest(c.cfg.GetSubscriptionURL(), queryParams)
	if err != nil {
		return nil, agenterrors.Wrap(ErrSubscriptionQuery, err.Error())
	}

	subscriptions := make([]uc.CatalogItemSubscription, 0)
	json.Unmarshal(resBody, &subscriptions)

	// build the Subscription from the UC ones
	subs := make([]CentralSubscription, 0)
	for i := range subscriptions {
		sub := CentralSubscription{
			CatalogItemSubscription: &subscriptions[i],
			apicClient:              c,
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// getAccessRequests -
func (c *ServiceClient) getAccessRequests(states []string) ([]AccessRequestSubscription, error) {
	queryParams := make(map[string]string)

	searchQuery := ""
	for _, state := range states {
		if searchQuery != "" {
			searchQuery += ","
		}
		searchQuery += "state.name==" + state
	}

	queryParams["query"] = searchQuery
	resBody, err := c.sendSubscriptionsRequest(c.cfg.GetAccessRequestsURL(), queryParams)
	if err != nil {
		return nil, err
	}

	subscriptions := make([]v1alpha1.AccessRequest, 0)
	json.Unmarshal(resBody, &subscriptions)

	// build the AccessRequestSubscription from the UC ones
	subs := make([]AccessRequestSubscription, 0)
	for i := range subscriptions {
		sub := AccessRequestSubscription{
			AccessRequest: &subscriptions[i],
			apicClient:    c,
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (c *ServiceClient) sendSubscriptionsRequest(url string, queryParams map[string]string) ([]byte, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		QueryParams: queryParams,
		Headers:     headers,
		Body:        nil,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, agenterrors.Wrap(ErrSubscriptionQuery, err.Error())
	}
	if response.Code != http.StatusOK && response.Code != http.StatusNotFound {
		readResponseErrors(response.Code, response.Body)
		return nil, ErrSubscriptionResp.FormatError(response.Code)
	}

	return response.Body, nil

}
