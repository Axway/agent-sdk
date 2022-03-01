package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
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

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.prov.CredentialProvision(&prov.Creds{})
	}

	if action == proto.Event_DELETED {
		h.prov.CredentialDeprovision(&prov.Creds{})
	}

	return nil
}
