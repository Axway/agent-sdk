package handler

import (
	"encoding/json"
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type credProv interface {
	CredentialProvision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential)
	CredentialDeprovision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus)
}

type credentials struct {
	prov   credProv
	client client
}

// NewCredentialHandler creates a Handler for Access Requests
func NewCredentialHandler(prov credProv, client client) Handler {
	return &credentials{
		prov:   prov,
		client: client,
	}
}

// Handle processes grpc events triggered for Credentials
func (h *credentials) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != mv1.CredentialGVK().Kind || h.prov == nil || action == proto.Event_SUBRESOURCEUPDATED {
		return nil
	}

	cr := &mv1.Credential{}
	err := cr.FromInstance(resource)
	if err != nil {
		return err
	}

	ok := isStatusFound(cr.Status)
	if !ok {
		return nil
	}

	if cr.Status.Level != statusPending {
		return nil
	}

	log.Infof("Received a %s event for a AccessRequest", action.String())
	bts, _ := json.MarshalIndent(cr, "", "\t")
	log.Info(string(bts))

	app, err := h.getManagedApp(cr)
	if err != nil {
		return err
	}

	credDetails := util.GetAgentDetails(cr)
	appDetails := util.GetAgentDetails(app)

	creds := &creds{
		appDetails:  appDetails,
		credDetails: credDetails,
		credType:    cr.Spec.CredentialRequestDefinition,
		managedApp:  cr.Spec.ManagedApplication,
		requestType: string(cr.Request),
	}

	if action == proto.Event_DELETED {
		log.Info("Deprovisioning the Credentials")
		h.prov.CredentialDeprovision(creds)
		return nil
	}

	var status prov.RequestStatus
	var credentialData prov.Credential

	if cr.Status.Level == statusPending {
		log.Info("Provisioning the Credentials")
		log.Infof("%+v", creds)
		status, credentialData = h.prov.CredentialProvision(creds)
		cr.Data = credentialData.GetData()
	}

	s := prov.NewStatusReason(status)
	cr.Status = &s

	details := util.MergeMapStringInterface(util.GetAgentDetails(cr), status.GetProperties())
	util.SetAgentDetails(cr, details)

	err = h.client.CreateSubResourceScoped(
		mv1.EnvironmentResourceName,
		cr.Metadata.Scope.Name,
		cr.PluralName(),
		cr.Name,
		cr.Group,
		cr.APIVersion,
		map[string]interface{}{
			defs.XAgentDetails: util.GetAgentDetails(cr),
			"status":           cr.Status,
			"data":             cr.Data,
		},
	)

	return err
}

func (h *credentials) getManagedApp(cred *mv1.Credential) (*v1.ResourceInstance, error) {
	url := fmt.Sprintf(
		"/management/v1alpha1/environments/%s/managedapplications/%s",
		cred.Metadata.Scope.Name,
		cred.Spec.ManagedApplication,
	)
	return h.client.GetResource(url)
}

type creds struct {
	managedApp  string
	credType    string
	requestType string
	credDetails map[string]interface{}
	appDetails  map[string]interface{}
}

func (c creds) GetApplicationName() string {
	return c.managedApp
}

func (c creds) GetCredentialType() string {
	return c.credType
}

// GetRequestType returns the type of request for the credentials
func (c creds) GetRequestType() string {
	return c.requestType
}

// GetCredentialDetailsValue returns a value found on the 'x-agent-details' sub resource of the Credentials.
func (c creds) GetCredentialDetailsValue(key string) interface{} {
	if c.credDetails == nil {
		return nil
	}
	return c.credDetails[key]
}

// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
func (c creds) GetApplicationDetailsValue(key string) interface{} {
	if c.appDetails == nil {
		return nil
	}
	return c.appDetails[key]
}
