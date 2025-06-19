package handler

import (
	"context"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/transaction/util"
	sdkUtil "github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type sampling interface {
	EnableSampling(samplingLimit int32, samplingEndTime time.Time, endpointsInfo map[string]management.TraceabilityAgentAgentstateSamplingEndpoints)
}

type agentCache interface {
	AddTeam(team *definitions.PlatformTeam)
	ListAPIServiceInstances() []*v1.ResourceInstance
}

type apicClient interface {
	GetTeam(map[string]string) ([]definitions.PlatformTeam, error)
	CreateSubResource(rm v1.ResourceMeta, subs map[string]interface{}) error
}

type TraceabilityTriggerHandler interface {
	TriggerTraceability()
}

type ComplianceAgentHandler interface {
	TriggerProcessing()
}

// Register an AgentResourceUpdateHandler in an agent to trigger events when changes to the resource is made
type AgentResourceUpdateHandler interface {
	AgentResourceUpdate(ctx context.Context, resource *v1.ResourceInstance)
}

type agentTypeHandler func(action proto.Event_Type, subres string, resource *v1.ResourceInstance) error

type agentResourceHandler struct {
	logger               log.FieldLogger
	agentResourceManager resource.Manager
	sampler              sampling
	agentTypeHandler     map[string]agentTypeHandler
	cache                agentCache
	apicClient           apicClient
}

// NewAgentResourceHandler - creates a Handler for Agent resources
func NewAgentResourceHandler(agentResourceManager resource.Manager, sampler sampling, cache agentCache, apiClient apicClient) Handler {
	h := &agentResourceHandler{
		logger:               log.NewFieldLogger().WithComponent("agentResourceHandler").WithPackage("handler"),
		agentResourceManager: agentResourceManager,
		sampler:              sampler,
		cache:                cache,
		apicClient:           apiClient,
	}
	h.agentTypeHandler = map[string]agentTypeHandler{
		management.DiscoveryAgentGVK().Kind:    h.handleDiscovery,
		management.TraceabilityAgentGVK().Kind: h.handleTraceability,
		management.ComplianceAgentGVK().Kind:   h.handleCompliance,
	}
	return h
}

func (h *agentResourceHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	// skip any processing if the agent resource manager is not set
	if h.agentResourceManager == nil {
		return nil
	}
	subres := ""
	if meta != nil {
		subres = meta.Subresource
	}

	action := GetActionFromContext(ctx)
	handlerFunc, ok := h.agentTypeHandler[resource.Kind]
	if !ok {
		return nil
	}
	if action == proto.Event_SUBRESOURCEUPDATED && subres == definitions.XAgentDetails {
		h.handleUpdateTrigger(resource)
	}
	handlerFunc(action, subres, resource)

	return nil
}

func (h *agentResourceHandler) handleUpdateTrigger(resource *v1.ResourceInstance) {
	agentDetails, ok := resource.GetSubResource(definitions.XAgentDetails).(map[string]interface{})
	if !ok {
		return
	}
	update, _ := agentDetails[definitions.TriggerTeamUpdate].(bool)
	if !update {
		return
	}

	agentDetails[definitions.TriggerTeamUpdate] = false
	subs := map[string]interface{}{definitions.XAgentDetails: agentDetails}
	if err := h.apicClient.CreateSubResource(resource.ResourceMeta, subs); err != nil {
		h.logger.WithError(err).WithField("name", resource.Name).Errorf("failed to reset the agent details triggerUpdate")
	}
	RefreshTeamCache(h.apicClient, h.cache)
}

func (h *agentResourceHandler) handleDiscovery(action proto.Event_Type, subres string, resource *v1.ResourceInstance) error {
	if action == proto.Event_UPDATED {
		h.agentResourceManager.SetAgentResource(resource)
	}
	return nil
}

func (h *agentResourceHandler) handleTraceability(action proto.Event_Type, subres string, resource *v1.ResourceInstance) error {

	switch {
	case action == proto.Event_UPDATED:
		h.agentResourceManager.SetAgentResource(resource)
	case action == proto.Event_SUBRESOURCEUPDATED && subres == management.TraceabilityAgentAgentstateSubResourceName:
		return h.handleTraceabilitySampling(resource)
	}

	return nil
}

func (h *agentResourceHandler) handleTraceabilitySampling(resource *v1.ResourceInstance) error {
	ta := &management.TraceabilityAgent{}
	err := ta.FromInstance(resource)
	if err != nil {
		return err
	}

	if ta.Agentstate.Sampling == nil || (!ta.Agentstate.Sampling.Enabled && len(ta.Agentstate.Sampling.Endpoints) == 0) {
		return nil
	}

	endpointsInfo := h.handleEndpointsSampling(ta.Agentstate.Sampling.Endpoints)
	h.sampler.EnableSampling(ta.Agentstate.Sampling.Limit, time.Time(ta.Agentstate.Sampling.EndTime), endpointsInfo)

	if traceabilityTriggerHandler, ok := h.agentResourceManager.GetHandler().(TraceabilityTriggerHandler); ok {
		traceabilityTriggerHandler.TriggerTraceability()
	}
	return nil
}

func (h *agentResourceHandler) handleCompliance(action proto.Event_Type, subres string, resource *v1.ResourceInstance) error {
	switch {
	case action == proto.Event_UPDATED:
		h.agentResourceManager.SetAgentResource(resource)
	case action == proto.Event_SUBRESOURCEUPDATED && subres == definitions.XAgentDetails:
		return h.handleComplianceProcessing(resource)
	}

	return nil
}

func (h *agentResourceHandler) handleComplianceProcessing(resource *v1.ResourceInstance) error {
	trigger, _ := sdkUtil.GetAgentDetailsValue(resource, definitions.ComplianceAgentTrigger)
	if complianceAgentHandler, ok := h.agentResourceManager.GetHandler().(ComplianceAgentHandler); ok && trigger == "true" {
		defer h.agentResourceManager.AddUpdateAgentDetails(definitions.ComplianceAgentTrigger, "false")
		complianceAgentHandler.TriggerProcessing()
	}
	return nil
}

func (h *agentResourceHandler) handleEndpointsSampling(endpoints []management.TraceabilityAgentAgentstateSamplingEndpoints) map[string]management.TraceabilityAgentAgentstateSamplingEndpoints {
	endpointsInfo := make(map[string]management.TraceabilityAgentAgentstateSamplingEndpoints) // apiID -> endpoints
	apiSIInfo := make(map[string]string)                                                      // basepath -> apiID

	apiSIRIs := h.cache.ListAPIServiceInstances()

	for _, apiSIRI := range apiSIRIs {
		apiSI := management.NewAPIServiceInstance("", "")
		err := apiSI.FromInstance(apiSIRI)
		if err != nil {
			h.logger.WithError(err).Errorf("failed to convert API Service Instance %s to management APIServiceInstance", apiSIRI.Metadata.ID)
			continue
		}

		apiID, err := sdkUtil.GetAgentDetailsValue(apiSI, definitions.AttrExternalAPIID)
		if err != nil || apiID == "" {
			h.logger.WithError(err).Warnf("API Service Instance %s does not have external API ID", apiSIRI.Metadata.ID)
			continue
		}
		apiID = strings.TrimPrefix(apiID, util.SummaryEventProxyIDPrefix)

		for _, endpoint := range apiSI.Spec.Endpoint {
			if _, ok := apiSIInfo[endpoint.Routing.BasePath]; ok {
				// if the basepath is already mapped to an apiID, skip this endpoint
				continue
			}
			apiSIInfo[endpoint.Routing.BasePath] = apiID
		}
	}

	for _, endpoint := range endpoints {
		if apiID, ok := apiSIInfo[endpoint.BasePath]; ok {
			endpointsInfo[apiID] = endpoint
		}
	}

	return endpointsInfo
}

func RefreshTeamCache(apicClient apicClient, cache agentCache) {
	platformTeams, err := apicClient.GetTeam(map[string]string{})
	if err != nil || len(platformTeams) == 0 {
		return
	}

	for _, team := range platformTeams {
		cache.AddTeam(&team)
	}
}
