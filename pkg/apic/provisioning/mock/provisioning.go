package mock

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
)

type MockApplicationRequest struct {
	provisioning.ApplicationRequest
	ID       string
	AppName  string
	Details  map[string]string
	TeamName string
}

func (m MockApplicationRequest) GetID() string {
	return m.ID
}

func (m MockApplicationRequest) GetManagedApplicationName() string {
	return m.AppName
}

func (m MockApplicationRequest) GetApplicationDetailsValue(key string) string {
	return m.Details[key]
}
func (m MockApplicationRequest) GetTeamName() string {
	return m.TeamName
}

type MockCredentialRequest struct {
	provisioning.CredentialRequest
	ID          string
	AppDetails  map[string]string
	AppName     string
	Name        string
	CredDefName string
	Details     map[string]string
	CredData    map[string]interface{}
	Action      provisioning.CredentialAction
}

func (m MockCredentialRequest) GetApplicationName() string {
	return m.AppName
}

func (m MockCredentialRequest) GetID() string {
	return m.ID
}

func (m MockCredentialRequest) GetName() string {
	return m.Name
}

func (m MockCredentialRequest) GetCredentialDetailsValue(key string) string {
	return m.Details[key]
}

func (m MockCredentialRequest) GetApplicationDetailsValue(key string) string {
	return m.AppDetails[key]
}

func (m MockCredentialRequest) GetCredentialType() string {
	return m.CredDefName
}

func (m MockCredentialRequest) GetCredentialData() map[string]interface{} {
	return m.CredData
}

func (m MockCredentialRequest) GetCredentialAction() provisioning.CredentialAction {
	return m.Action
}

func (m MockCredentialRequest) IsIDPCredential() bool {
	return false
}

func GetIDPProvider() oauth.Provider {
	return nil
}

func GetIDPCredentialData() provisioning.IDPCredentialData {
	return nil
}

type MockAccessRequest struct {
	provisioning.AccessRequest
	ID                            string
	AppDetails                    map[string]string
	AppName                       string
	Details                       map[string]string
	InstanceDetails               map[string]interface{}
	AccessRequestData             map[string]interface{}
	AccessRequestProvisioningData interface{}
	QuotaLimit                    int64
	QuotaInterval                 provisioning.QuotaInterval
	PlanName                      string
}

func (m MockAccessRequest) GetID() string {
	return m.ID
}

func (m MockAccessRequest) GetAccessRequestData() map[string]interface{} {
	return m.AccessRequestData
}

func (m MockAccessRequest) GetAccessRequestProvisioningData() interface{} {
	return m.AccessRequestProvisioningData
}

func (m MockAccessRequest) GetApplicationName() string {
	return m.AppName
}

func (m MockAccessRequest) GetAccessRequestDetailsValue(key string) string {
	return m.Details[key]
}

func (m MockAccessRequest) GetApplicationDetailsValue(key string) string {
	return m.AppDetails[key]
}

func (m MockAccessRequest) GetInstanceDetails() map[string]interface{} {
	return m.InstanceDetails
}

func (m MockAccessRequest) GetQuota() provisioning.Quota {
	if m.QuotaInterval == 0 {
		return nil
	}
	return m
}

func (m MockAccessRequest) GetPlanName() string {
	return m.PlanName
}

func (m MockAccessRequest) GetLimit() int64 {
	return m.QuotaLimit
}

func (m MockAccessRequest) GetInterval() provisioning.QuotaInterval {
	return m.QuotaInterval
}

func (m MockAccessRequest) GetIntervalString() string {
	return m.QuotaInterval.String()
}

type MockRequestStatus struct {
	Msg        string
	Properties map[string]string
	Status     provisioning.Status
	Reasons    []v1.ResourceStatusReason
}

func (m MockRequestStatus) GetStatus() provisioning.Status {
	return m.Status
}

func (m MockRequestStatus) GetMessage() string {
	return m.Msg
}

func (m MockRequestStatus) GetProperties() map[string]string {
	return m.Properties
}

// GetStatus returns the Status level
func (m MockRequestStatus) GetReasons() []v1.ResourceStatusReason {
	return m.Reasons
}
