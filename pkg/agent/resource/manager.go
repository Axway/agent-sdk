package resource

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// QA EnvVars
const qaTriggerSevenDayRefreshCache = "QA_CENTRAL_TRIGGER_REFRESH_CACHE"

type EventSyncCache interface {
	RebuildCache()
}

// Manager - interface to manage agent resource
type Manager interface {
	OnConfigChange(cfg config.CentralConfig, apicClient apic.Client)
	GetAgentResource() *apiv1.ResourceInstance
	SetAgentResource(agentResource *apiv1.ResourceInstance)
	FetchAgentResource() error
	UpdateAgentStatus(status, prevStatus, message string) error
	AddUpdateAgentDetails(key, value string)
	SetRebuildCacheFunc(rebuildCache EventSyncCache)
	RegisterHandler(handler interface{})
	GetHandler() interface{}
}

type executeAPIClient interface {
	CreateSubResource(rm v1.ResourceMeta, subs map[string]interface{}) error
	GetResource(url string) (*v1.ResourceInstance, error)
	CreateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	UpdateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
}

type agentResourceManager struct {
	agentResource              *apiv1.ResourceInstance
	handler                    interface{}
	prevAgentResHash           uint64
	apicClient                 executeAPIClient
	cfg                        config.CentralConfig
	agentResourceChangeHandler func()
	agentDetails               map[string]interface{}
	logger                     log.FieldLogger
	rebuildCache               EventSyncCache
}

// NewAgentResourceManager - Create a new agent resource manager
func NewAgentResourceManager(cfg config.CentralConfig, apicClient executeAPIClient, agentResourceChangeHandler func()) (Manager, error) {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent").
		WithComponent("agentResourceManager")
	m := &agentResourceManager{
		cfg:                        cfg,
		apicClient:                 apicClient,
		agentResourceChangeHandler: agentResourceChangeHandler,
		agentDetails:               make(map[string]interface{}),
		logger:                     logger,
	}

	if m.getAgentResourceType() != nil {
		err := m.FetchAgentResource()
		if err != nil {
			return nil, err
		}
	} else if m.cfg.GetAgentName() != "" {
		return nil, errors.Wrap(apic.ErrCentralConfig, "Agent name cannot be set. Config is used only for agents with API server resource definition")
	}
	return m, nil
}

// OnConfigChange - Applies central config change to the manager
func (a *agentResourceManager) OnConfigChange(cfg config.CentralConfig, apicClient apic.Client) {
	a.apicClient = apicClient
	a.cfg = cfg
}

// GetAgentResource - Returns the agent resource
func (a *agentResourceManager) GetAgentResource() *apiv1.ResourceInstance {
	return a.agentResource
}

// SetAgentResource - Sets the agent resource which triggers agent resource change handler
func (a *agentResourceManager) SetAgentResource(agentResource *apiv1.ResourceInstance) {
	if agentResource != nil && agentResource.Name == a.cfg.GetAgentName() {
		a.agentResource = agentResource
		a.onResourceChange()
	}
}

func (a *agentResourceManager) SetRebuildCacheFunc(rebuildCache EventSyncCache) {
	a.rebuildCache = rebuildCache
}

// FetchAgentResource - Gets the agent resource using API call to apiserver
func (a *agentResourceManager) FetchAgentResource() error {
	if a.cfg.GetAgentName() == "" {
		a.logger.Trace("skipping to fetch agent resource, agent name not configured")
		return nil
	}

	var err error
	a.agentResource, err = a.getAgentResource()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			a.agentResource, err = a.createAgentResource()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		a.checkAgentResource()
	}

	a.onResourceChange()
	return nil
}

// UpdateAgentStatus - Updates the agent status in agent resource
func (a *agentResourceManager) UpdateAgentStatus(status, prevStatus, message string) error {
	if a.cfg == nil || a.cfg.GetAgentName() == "" {
		a.logger.WithField("status", status).
			WithField("previousStatus", prevStatus).
			Trace("skipping agent status update, agent name config not set")
		return nil
	}

	agentInstance := a.agentResource
	if a.agentResource == nil {
		agentInstance = a.getAgentResourceType()
	}
	if a.agentResource == nil && agentInstance != nil {
		a.logger.Info("re-initializing agent resource")
		err := a.FetchAgentResource()
		if err != nil {
			return err
		}
	}

	if a.agentResource == nil {
		a.logger.WithField("status", status).
			WithField("previousStatus", prevStatus).
			Trace("skipping agent status update, agent resource not initialized")
		return nil
	}

	statusSubResourceName := management.DiscoveryAgentStatusSubResourceName
	// using discovery agent status here, but all agent status resources have the same structure
	agentStatus := newDAStatus(agentInstance.ResourceMeta, status, prevStatus, message)

	// See if we need to rebuildCache
	timeToRebuild, _ := a.shouldRebuildCache()
	if timeToRebuild && a.rebuildCache != nil {
		a.rebuildCache.RebuildCache()
	}

	subResources := make(map[string]interface{})
	subResources[statusSubResourceName] = agentStatus
	// add any details
	if len(a.agentDetails) > 0 {
		util.SetAgentDetails(agentInstance, a.agentDetails)
		subResources[definitions.XAgentDetails] = agentInstance.SubResources[definitions.XAgentDetails]
	}

	err := a.apicClient.CreateSubResource(agentInstance.ResourceMeta, subResources)
	return err
}

// 1. On UpdateAgentStatus, if x-agent-details, key "cacheUpdateTime" doesn't exist or empty, rebuild cache to populate cacheUpdateTime
// 2. On UpdateAgentStatus, if x-agent-details exists, check to see if its past 7 days since rebuildCache was ran.  If its pass 7 days, rebuildCache
func (a *agentResourceManager) shouldRebuildCache() (bool, error) {
	rebuildCache := false
	agentInstance := a.GetAgentResource()
	agentDetails := agentInstance.GetSubResource(definitions.XAgentDetails)

	if agentDetails == nil {
		// x-agent-details hasn't been established yet. Rebuild cache to populate cacheUpdateTime
		a.logger.Trace("create x-agent-detail subresource and add key 'cacheUpdateTime'")
		rebuildCache = true
	} else {
		value, exists := agentDetails.(map[string]interface{})["cacheUpdateTime"]
		if value != nil {
			logger := a.logger.WithField("cacheUpdateTime", value)
			// get current cacheUpdateTime from x-agent-details
			convToTimestamp, err := strconv.ParseInt(value.(string), 10, 64)
			if err != nil {
				logger.WithError(err).Error("unable to parse cache update time")
				return false, err
			}
			currentCacheUpdateTime := time.Unix(0, convToTimestamp)
			logger.Trace("the current scheduled refresh cache date - %s", time.Unix(0, currentCacheUpdateTime.UnixNano()).Format("2006-01-02 15:04:05.000000"))

			// check to see if 7 days have passed since last refresh cache. currentCacheUpdateTime is the date at the time we rebuilt cache plus 7 days(in event sync - RebuildCache)
			if a.getCurrentTime() > currentCacheUpdateTime.UnixNano() {
				logger.Trace("the current date is greater than the current scheduled refresh date - time to rebuild cache")
				rebuildCache = true
			}
		} else {
			if !exists {
				// x-agent-details exists, however, cacheUpdateTime key doesn't exist. Rebuild cache to populate cacheUpdateTime
				a.logger.Trace("update x-agent-detail subresource and add key 'cacheUpdateTime'")
				rebuildCache = true
			}
		}
	}

	return rebuildCache, nil
}

func (a *agentResourceManager) getCurrentTime() int64 {
	val := os.Getenv(qaTriggerSevenDayRefreshCache)
	if val == "" {
		// if this isn't set, then just pass back the current time
		return time.Now().UnixNano()
	}
	// if this is set, then pass back the current time, plus 7 days to trigger a rebuild
	return time.Now().Add(7 * 24 * time.Hour).UnixNano()
}

// GetAgentDetails - Gets current agent details
func (a *agentResourceManager) GetAgentDetails() map[string]interface{} {
	return a.agentDetails
}

// AddUpdateAgentDetails - Adds a new or Updates an existing key on the agent details sub resource
func (a *agentResourceManager) AddUpdateAgentDetails(key, value string) {
	a.agentDetails[key] = value
}

// getTimestamp - Returns current timestamp formatted for API Server
func getTimestamp() apiv1.Time {
	activityTime := time.Now()
	newV1Time := apiv1.Time(activityTime)
	return newV1Time
}

func newDAStatus(rm apiv1.ResourceMeta, status, prevStatus, message string) management.DiscoveryAgentStatus {
	now := getTimestamp()
	daStatus := management.DiscoveryAgentStatus{
		Version:                strings.Split(config.AgentVersion, "-")[0], // report just the semver version
		LatestAvailableVersion: config.AgentLatestVersion,
		State:                  status,
		PreviousState:          prevStatus,
		Message:                message,
		LastActivityTime:       now,
		SdkVersion:             config.SDKVersion,
	}
	statusIface := rm.GetSubResource(management.DiscoveryAgentStatusSubResourceName)
	if statusIface == nil {
		return daStatus
	}

	existingStatus, ok := statusIface.(management.DiscoveryAgentStatus)
	if !ok {
		bts, err := json.Marshal(statusIface)
		if err != nil {
			return daStatus
		}
		if err = json.Unmarshal(bts, &existingStatus); err != nil {
			return daStatus
		}
	}
	if existingStatus.State != "" {
		daStatus.PreviousState = existingStatus.State
	}

	// maybe add config for this if everything else is fine
	if time.Time(now).Sub(time.Time(existingStatus.LastActivityTime)) > 6*time.Hour {
		daStatus.LastActivityTime = now
		return daStatus
	}

	daStatus.LastActivityTime = existingStatus.LastActivityTime
	return daStatus
}

func applyResConfigToCentralConfig(cfg *config.CentralConfiguration, resCfgAdditionalTags, resCfgTeamID, resCfgLogLevel string, managedEnvs []string) {
	agentResLogLevel := log.GlobalLoggerConfig.GetLevel()
	if resCfgLogLevel != "" && !strings.EqualFold(agentResLogLevel, resCfgLogLevel) {
		log.GlobalLoggerConfig.Level(resCfgLogLevel).Apply()
	}

	if resCfgAdditionalTags != "" && !strings.EqualFold(cfg.TagsToPublish, resCfgAdditionalTags) {
		cfg.TagsToPublish = resCfgAdditionalTags
	}

	// If config team is blank, check resource team name.  If resource team name is not blank, use resource team name
	if resCfgTeamID != "" && cfg.TeamName == "" {
		cfg.SetTeamID(resCfgTeamID)
	}

	cfg.SetManagedEnvironments(managedEnvs)
}

func (a *agentResourceManager) onResourceChange() {
	if a.agentResource != nil {
		subRes := a.agentResource.GetSubResource(definitions.XAgentDetails)
		if details, ok := subRes.(map[string]interface{}); ok {
			a.agentDetails = details
		}
	}

	isChanged := (a.prevAgentResHash != 0)
	agentResHash, _ := util.ComputeHash(a.agentResource)
	if a.prevAgentResHash != 0 && a.prevAgentResHash == agentResHash {
		isChanged = false
	}
	a.prevAgentResHash = agentResHash
	if isChanged {
		// merge agent resource config with central config
		a.mergeResourceWithConfig()
		if a.agentResourceChangeHandler != nil {
			a.agentResourceChangeHandler()
		}
	}
}

func (a *agentResourceManager) getAgentResourceType() *v1.ResourceInstance {
	var agentRes v1.Interface
	switch a.cfg.GetAgentType() {
	case config.DiscoveryAgent:
		res := management.NewDiscoveryAgent(a.cfg.GetAgentName(), a.cfg.GetEnvironmentName())
		res.Spec.DataplaneType = config.AgentDataPlaneType
		agentRes = res
	case config.TraceabilityAgent:
		res := management.NewTraceabilityAgent(a.cfg.GetAgentName(), a.cfg.GetEnvironmentName())
		res.Spec.DataplaneType = config.AgentDataPlaneType
		agentRes = res
	case config.ComplianceAgent:
		res := management.NewComplianceAgent(a.cfg.GetAgentName(), a.cfg.GetEnvironmentName())
		res.Spec.DataplaneType = config.AgentDataPlaneType
		agentRes = res
	}
	var agentInstance *v1.ResourceInstance
	if agentRes != nil {
		agentInstance, _ = agentRes.AsInstance()
	}
	return agentInstance
}

// GetAgentResource - returns the agent resource
func (a *agentResourceManager) createAgentResource() (*v1.ResourceInstance, error) {
	agentRes := a.getAgentResourceType()
	if agentRes == nil {
		return nil, fmt.Errorf("unknown agent type")
	}
	a.logger.
		WithField("scope", agentRes.Metadata.Scope).
		WithField("kind", agentRes.Kind).
		WithField("name", agentRes.Name).
		Info("creating agent resource")
	return a.apicClient.CreateResourceInstance(agentRes)
}

// GetAgentResource - returns the agent resource
func (a *agentResourceManager) checkAgentResource() (*v1.ResourceInstance, error) {
	var agentRes v1.Interface
	logger := a.logger.WithField("scope", a.agentResource.Metadata.Scope).WithField("kind", a.agentResource.Kind).WithField("name", a.agentResource.Name)

	currDataplaneType := apic.Unidentified.String()
	if a.agentResource.Kind == management.DiscoveryAgentGVK().Kind {
		da := management.NewDiscoveryAgent("", "")
		da.FromInstance(a.agentResource)
		currDataplaneType = da.Spec.DataplaneType
		da.Spec.DataplaneType = config.AgentDataPlaneType
		agentRes = da
	} else if a.agentResource.Kind == management.TraceabilityAgentGVK().Kind {
		ta := management.NewTraceabilityAgent("", "")
		ta.FromInstance(a.agentResource)
		currDataplaneType = ta.Spec.DataplaneType
		ta.Spec.DataplaneType = config.AgentDataPlaneType
		agentRes = ta
	} else if a.agentResource.Kind == management.ComplianceAgentGVK().Kind {
		ca := management.NewComplianceAgent("", "")
		ca.FromInstance(a.agentResource)
		currDataplaneType = ca.Spec.DataplaneType
		ca.Spec.DataplaneType = config.AgentDataPlaneType
		agentRes = ca
	}

	// nothing to update
	if currDataplaneType == config.AgentDataPlaneType {
		return a.agentResource, nil
	}

	logger.Info("updating agent resource")
	return a.apicClient.UpdateResourceInstance(agentRes)
}

// GetAgentResource - returns the agent resource
func (a *agentResourceManager) getAgentResource() (*v1.ResourceInstance, error) {
	agentRes := a.getAgentResourceType()
	if agentRes == nil {
		return nil, fmt.Errorf("unknown agent type")
	}

	return a.apicClient.GetResource(agentRes.GetSelfLink())
}

func (a *agentResourceManager) mergeResourceWithConfig() {
	// IMP - To be removed once the model is in production
	if a.cfg.GetAgentName() == "" {
		return
	}

	switch a.getAgentResourceType().Kind {
	case management.DiscoveryAgentGVK().Kind:
		mergeDiscoveryAgentWithConfig(a.GetAgentResource(), a.cfg.(*config.CentralConfiguration))
	case management.TraceabilityAgentGVK().Kind:
		mergeTraceabilityAgentWithConfig(a.GetAgentResource(), a.cfg.(*config.CentralConfiguration))
	case management.ComplianceAgentGVK().Kind:
		mergeComplianceAgentWithConfig(a.GetAgentResource(), a.cfg.(*config.CentralConfiguration))
	default:
		panic(ErrUnsupportedAgentType)
	}
}

func (a *agentResourceManager) RegisterHandler(handler interface{}) {
	a.handler = handler
}

func (a *agentResourceManager) GetHandler() interface{} {
	return a.handler
}
