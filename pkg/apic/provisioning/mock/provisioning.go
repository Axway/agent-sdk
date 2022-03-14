package mock

import "github.com/Axway/agent-sdk/pkg/apic/provisioning"

type MockApplicationRequest struct {
	provisioning.ApplicationRequest
	AppName  string
	TeamName string
	Details  map[string]interface{}
}

func (m MockApplicationRequest) GetManagedApplicationName() string { return m.AppName }
func (m MockApplicationRequest) GetApplicationDetailsValue(key string) interface{} {
	return m.Details[key]
}
func (m MockApplicationRequest) GetTeamName() string { return m.TeamName }

type MockCredentialRequest struct {
	provisioning.CredentialRequest
	AppName    string
	CredType   string
	Details    map[string]interface{}
	AppDetails map[string]interface{}
}

func (m MockCredentialRequest) GetApplicationName() string { return m.AppName }
func (m MockCredentialRequest) GetCredentialDetailsValue(key string) interface{} {
	return m.Details[key]
}
func (m MockCredentialRequest) GetApplicationDetailsValue(key string) interface{} {
	return m.AppDetails[key]
}
func (m MockCredentialRequest) GetCredentialType() string { return m.CredType }

type MockAccessRequest struct {
	provisioning.CredentialRequest
	AppName    string
	APIID      string
	Stage      string
	Details    map[string]interface{}
	AppDetails map[string]interface{}
}

func (m MockAccessRequest) GetApplicationName() string { return m.AppName }
func (m MockAccessRequest) GetAccessRequestDetailsValue(key string) interface{} {
	return m.Details[key]
}
func (m MockAccessRequest) GetApplicationDetailsValue(key string) interface{} {
	return m.AppDetails[key]
}
func (m MockAccessRequest) GetAPIID() string { return m.APIID }
func (m MockAccessRequest) GetStage() string { return m.Stage }
