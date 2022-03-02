package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const credentialKind = "Credential"

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
	if resource.Kind != credentialKind {
		return nil
	}

	ar := &mv1.AccessRequest{}
	err := ar.FromInstance(resource)
	if err != nil {
		return err
	}

	creds := &Creds{}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.prov.CredentialProvision(creds)
	}

	if action == proto.Event_DELETED {
		h.prov.CredentialDeprovision(creds)
	}

	return nil
}

type Creds struct {
}

func (c Creds) GetApplicationName() string {
	return "app name"
}

func (c Creds) GetCredentialType() prov.CredentialType {
	return prov.APIKeyCredential
}

func (c Creds) GetRequestType() string {
	return "request type"
}

func (c Creds) GetProperty(key string) string {
	return "prop"
}
