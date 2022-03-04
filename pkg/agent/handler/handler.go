package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// Handler interface used by the EventListener to process events.
type Handler interface {
	// Handle receives the type of the event (add, update, delete), event metadata and the API Server resource, if it exists.
	Handle(action proto.Event_Type, eventMetadata *proto.EventMeta, resource *v1.ResourceInstance) error
}

// client is an interface that is implemented by the ServiceClient in apic/client.go.
type client interface {
	GetResource(url string) (*v1.ResourceInstance, error)
	CreateResource(url string, bts []byte) (*v1.ResourceInstance, error)
	UpdateResource(url string, bts []byte) (*v1.ResourceInstance, error)
	CreateSubResourceScoped(scopeKindPlural, scopeName, resKindPlural, name, group, version string, subs map[string]interface{}) error
}

func isStatusFound(rs *v1.ResourceStatus) bool {
	if rs == nil || rs.Level == "" {
		return false
	}
	return true
}

type FakeProvisioner struct {
}

func (f FakeProvisioner) ApplicationRequestProvision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

func (f FakeProvisioner) ApplicationRequestDeprovision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

func (f FakeProvisioner) AccessRequestProvision(accessRequest prov.AccessRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

func (f FakeProvisioner) AccessRequestDeprovision(accessRequest prov.AccessRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

func (f FakeProvisioner) CredentialProvision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential) {
	return &FakeStatus{}, &FakeCredential{}
}

func (f FakeProvisioner) CredentialDeprovision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

type FakeStatus struct {
}

func (f FakeStatus) GetStatus() prov.Status {
	return prov.Success
}

func (f FakeStatus) GetMessage() string {
	return "message"
}

func (f FakeStatus) GetProperties() map[string]interface{} {
	return map[string]interface{}{
		"status_key": "status_val",
	}
}

type FakeCredential struct{}

func (f FakeCredential) GetData() map[string]interface{} {
	return map[string]interface{}{
		"credential_key": "credential_value",
	}
}
