package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type credentialProvision interface {
	CredentialProvision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential)
	CredentialDeprovision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus)
}

type credentials struct {
	prov credentialProvision
}

// NewcredentialHandler creates a Handler for Access Requests
func NewcredentialHandler() Handler {
	return &credentials{}
}

func (h *credentials) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != mv1.CredentialGVK().Kind {
		return nil
	}

	ar := &mv1.AccessRequest{}
	err := ar.FromInstance(resource)
	if err != nil {
		return err
	}

	creds := &creds{}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.prov.CredentialProvision(creds)
	}

	if action == proto.Event_DELETED {
		h.prov.CredentialDeprovision(creds)
	}

	return nil
}

type creds struct {
	apiID       string
	appDetails  map[string]interface{}
	credDetails map[string]interface{}
	managedApp  string
	credType    prov.CredentialType
	reqType     prov.RequestType
}

func (c creds) GetApplicationName() string {
	return c.managedApp
}

func (c creds) GetCredentialType() prov.CredentialType {
	return c.credType
}

// GetRequestType returns the type of request for the credentials
func (c creds) GetRequestType() string {
	return c.reqType.String()
}

// GetCredentialDetails returns a value found on the 'x-agent-details' sub resource of the Credentials.
func (c creds) GetCredentialDetails(key string) interface{} {
	if c.credDetails == nil {
		return nil
	}
	return c.credDetails[key]
}

// GetApplicationDetails returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
func (c creds) GetApplicationDetails(key string) interface{} {
	if c.appDetails == nil {
		return nil
	}
	return c.appDetails[key]
}
