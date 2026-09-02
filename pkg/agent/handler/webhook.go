package handler

import (
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
)

// webhookDispatchDetailKey is the x-agent-details key that records which operation ("provision" or
// "deprovision") the agent last dispatched to the provisioning webhook for this resource. Reusing
// x-agent-details (rather than a new subresource) is safe here because the agent is the only writer of this
// key - the client's webhook, per docs/discovery/provisioning-webhook.md step 2, is scoped to write only the
// separate x-webhook-details subresource, so there's no risk of it clobbering this marker.
const webhookDispatchDetailKey = "provisioningWebhookDispatched"

const (
	webhookOperationProvision   = "provision"
	webhookOperationDeprovision = "deprovision"
)

// subResourceCarrier is satisfied by any apiserver resource instance type (ManagedApplication, AccessRequest,
// Credential, ...) - matches the generic GetSubResource/SetSubResource pair they all get from ResourceMeta,
// which is also what the unexported util.handler interface requires.
type subResourceCarrier interface {
	GetSubResource(key string) interface{}
	SetSubResource(key string, resource interface{})
}

// webhookDispatchedFor returns true if the resource's x-agent-details already record that operation was
// dispatched to the provisioning webhook - used to avoid re-dispatching on event redelivery (Central may
// redeliver an event for a resource for reasons unrelated to anything this handler wrote).
func webhookDispatchedFor(h subResourceCarrier, operation string) bool {
	v, _ := util.GetAgentDetailsValue(h, webhookDispatchDetailKey)
	return v == operation
}

// markWebhookDispatched records, in the resource's x-agent-details, that operation was just dispatched to
// the provisioning webhook. No resource status is written for this - the resource is left exactly as it was
// (still Pending, or still Deleting) until something (not yet built) reports real completion; see
// docs/discovery/provisioning-webhook.md.
func markWebhookDispatched(h subResourceCarrier, operation string) {
	_ = util.SetAgentDetailsKey(h, webhookDispatchDetailKey, operation)
}

// mirrorWebhookDetails copies the resource's x-webhook-details subresource into its x-agent-details, so
// existing code (traceability lookups, etc.) that only knows how to read x-agent-details keeps working once
// a provisioning webhook is configured - the webhook itself is scoped to write only x-webhook-details, not
// x-agent-details directly. Returns false if there is nothing to mirror.
func mirrorWebhookDetails(h subResourceCarrier) bool {
	webhookDetails, ok := h.GetSubResource(defs.XWebhookDetails).(map[string]interface{})
	if !ok || len(webhookDetails) == 0 {
		return false
	}

	agentDetails := util.GetAgentDetails(h)
	if agentDetails == nil {
		agentDetails = map[string]interface{}{}
	}
	for k, v := range webhookDetails {
		agentDetails[k] = v
	}
	util.SetAgentDetails(h, agentDetails)
	return true
}

// webhookApplicationRequest is the payload sent to the configured provisioning webhook for a ManagedApplication event
type webhookApplicationRequest struct {
	Operation              string                 `json:"operation"`
	ID                     string                 `json:"id"`
	ManagedApplicationName string                 `json:"managedApplicationName"`
	TeamName               string                 `json:"teamName"`
	ConsumerOrgID          string                 `json:"consumerOrgId,omitempty"`
	AgentDetails           map[string]interface{} `json:"agentDetails,omitempty"`
}

func newWebhookApplicationRequest(operation string, a provManagedApp) webhookApplicationRequest {
	return webhookApplicationRequest{
		Operation:              operation,
		ID:                     a.id,
		ManagedApplicationName: a.managedAppName,
		TeamName:               a.teamName,
		ConsumerOrgID:          a.consumerOrgID,
		AgentDetails:           a.data,
	}
}

// webhookQuota mirrors the fields of prov.Quota needed by the webhook contract
type webhookQuota struct {
	Limit    int64  `json:"limit"`
	Interval string `json:"interval"`
}

// webhookAccessRequest is the payload sent to the configured provisioning webhook for an AccessRequest event
type webhookAccessRequest struct {
	Operation               string                 `json:"operation"`
	ID                      string                 `json:"id"`
	ReferencedID            string                 `json:"referencedId,omitempty"`
	ManagedApplicationName  string                 `json:"managedApplicationName"`
	IsTransferring          bool                   `json:"isTransferring"`
	RequestData             map[string]interface{} `json:"requestData,omitempty"`
	ProvisioningData        interface{}            `json:"provisioningData,omitempty"`
	AccessDetails           map[string]interface{} `json:"accessDetails,omitempty"`
	ReferencedAccessDetails map[string]interface{} `json:"referencedAccessDetails,omitempty"`
	ApplicationDetails      map[string]interface{} `json:"applicationDetails,omitempty"`
	InstanceDetails         map[string]interface{} `json:"instanceDetails,omitempty"`
	Quota                   *webhookQuota          `json:"quota,omitempty"`
}

func newWebhookAccessRequest(operation string, r provAccReq) webhookAccessRequest {
	req := webhookAccessRequest{
		Operation:               operation,
		ID:                      r.id,
		ReferencedID:            r.refID,
		ManagedApplicationName:  r.managedApp,
		IsTransferring:          r.refID != "",
		RequestData:             r.requestData,
		ProvisioningData:        r.provData,
		AccessDetails:           r.accessDetails,
		ReferencedAccessDetails: r.refAccessDetails,
		ApplicationDetails:      r.appDetails,
		InstanceDetails:         r.instanceDetails,
	}
	if r.quota != nil {
		req.Quota = &webhookQuota{Limit: r.quota.GetLimit(), Interval: r.quota.GetIntervalString()}
	}
	return req
}

// webhookCredentialRequest is the payload sent to the configured provisioning webhook for a Credential event
type webhookCredentialRequest struct {
	Operation                 string                 `json:"operation"`
	ID                        string                 `json:"id"`
	Name                      string                 `json:"name"`
	ManagedApplicationName    string                 `json:"managedApplicationName"`
	CredentialType            string                 `json:"credentialType"`
	CredentialAction          int                    `json:"credentialAction"`
	CredentialData            map[string]interface{} `json:"credentialData,omitempty"`
	CredentialDetails         map[string]interface{} `json:"credentialDetails,omitempty"`
	ApplicationDetails        map[string]interface{} `json:"applicationDetails,omitempty"`
	CredentialSchema          map[string]interface{} `json:"credentialSchema,omitempty"`
	CredentialProvisionSchema map[string]interface{} `json:"credentialProvisionSchema,omitempty"`
	CredentialSchemaDetails   map[string]interface{} `json:"credentialSchemaDetails,omitempty"`
	ProvisionMode             string                 `json:"provisionMode,omitempty"`
	ExpirationDays            int                    `json:"expirationDays,omitempty"`
	IDPClientID               string                 `json:"idpClientId,omitempty"`
	IDPTokenEndpoint          string                 `json:"idpTokenEndpoint,omitempty"`
}

func newWebhookCredentialRequest(operation string, c *provCreds) webhookCredentialRequest {
	req := webhookCredentialRequest{
		Operation:                 operation,
		ID:                        c.id,
		Name:                      c.name,
		ManagedApplicationName:    c.managedApp,
		CredentialType:            c.credType,
		CredentialAction:          int(c.credAction),
		CredentialData:            c.credData,
		CredentialDetails:         c.credDetails,
		ApplicationDetails:        c.appDetails,
		CredentialSchema:          c.credSchema,
		CredentialProvisionSchema: c.credProvSchema,
		CredentialSchemaDetails:   c.credSchemaDetails,
		ProvisionMode:             c.provisionMode,
		ExpirationDays:            c.days,
	}
	if c.IsIDPCredential() {
		req.IDPClientID = c.GetIDPCredentialData().GetClientID()
		req.IDPTokenEndpoint = c.GetIDPProvider().GetTokenEndpoint()
	}
	return req
}
