package mock

import "github.com/Axway/agent-sdk/pkg/apic/provisioning"

type MockApplicationRequest struct {
	provisioning.ApplicationRequest
	AppName  string
	TeamName string
	Details  map[string]string
}

func (m MockApplicationRequest) GetManagedApplicationName() string { return m.AppName }
func (m MockApplicationRequest) GetApplicationDetailsValue(key string) string {
	return m.Details[key]
}
func (m MockApplicationRequest) GetTeamName() string { return m.TeamName }

type MockCredentialRequest struct {
	provisioning.CredentialRequest
	AppName    string
	CredType   string
	Details    map[string]string
	AppDetails map[string]string
}

func (m MockCredentialRequest) GetApplicationName() string { return m.AppName }
func (m MockCredentialRequest) GetCredentialDetailsValue(key string) string {
	return m.Details[key]
}
func (m MockCredentialRequest) GetApplicationDetailsValue(key string) string {
	return m.AppDetails[key]
}
func (m MockCredentialRequest) GetCredentialType() string { return m.CredType }

type MockAccessRequest struct {
	provisioning.CredentialRequest
	AppName    string
	APIID      string
	Stage      string
	Details    map[string]string
	AppDetails map[string]string
}

func (m MockAccessRequest) GetApplicationName() string { return m.AppName }
func (m MockAccessRequest) GetAccessRequestDetailsValue(key string) string {
	return m.Details[key]
}
func (m MockAccessRequest) GetApplicationDetailsValue(key string) string {
	return m.AppDetails[key]
}
func (m MockAccessRequest) GetAPIID() string { return m.APIID }
func (m MockAccessRequest) GetStage() string { return m.Stage }

type MockRequestStatus struct {
	Status     provisioning.Status
	Msg        string
	Properties map[string]string
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
