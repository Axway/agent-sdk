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
	CreateSubResourceScoped(scopeKindPlural, resKindPlural string, rm v1.ResourceMeta, subs map[string]interface{}) error
}

func isStatusFound(rs *v1.ResourceStatus) bool {
	if rs == nil || rs.Level == "" {
		return false
	}
	return true
}

func isNotStatusSubResourceUpdate(action proto.Event_Type, meta *proto.EventMeta) bool {
	if meta != nil {
		return (action == proto.Event_SUBRESOURCEUPDATED && meta.Subresource != "status")
	}
	return false
}

// FakeProvisioner -
type FakeProvisioner struct {
}

// ApplicationRequestProvision -
func (f FakeProvisioner) ApplicationRequestProvision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

// ApplicationRequestDeprovision -
func (f FakeProvisioner) ApplicationRequestDeprovision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

// AccessRequestProvision -
func (f FakeProvisioner) AccessRequestProvision(accessRequest prov.AccessRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

// AccessRequestDeprovision -
func (f FakeProvisioner) AccessRequestDeprovision(accessRequest prov.AccessRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

// CredentialProvision -
func (f FakeProvisioner) CredentialProvision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential) {
	return &FakeStatus{}, &FakeCredential{}
}

// CredentialDeprovision -
func (f FakeProvisioner) CredentialDeprovision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus) {
	return &FakeStatus{}
}

// FakeStatus -
type FakeStatus struct {
}

// GetStatus -
func (f FakeStatus) GetStatus() prov.Status {
	return prov.Success
}

// GetMessage -
func (f FakeStatus) GetMessage() string {
	return "message"
}

// GetProperties -
func (f FakeStatus) GetProperties() map[string]interface{} {
	return map[string]interface{}{
		"status_key": "status_val",
	}
}

// FakeCredential -
type FakeCredential struct{}

// GetData -
func (f FakeCredential) GetData() map[string]interface{} {
	return map[string]interface{}{
		"credential_key": "credential_value",
	}
}
