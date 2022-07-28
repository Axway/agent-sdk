package resource

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Manager - interface to manage agent resource
type Manager interface {
	OnConfigChange(cfg config.CentralConfig, apicClient apic.Client)
	GetAgentResource() *apiv1.ResourceInstance
	SetAgentResource(agentResource *apiv1.ResourceInstance)
	FetchAgentResource() error
	UpdateAgentStatus(status, prevStatus, message string) error
	AddUpdateAgentDetails(key, value string)
}

type executeAPIClient interface {
	CreateSubResource(rm v1.ResourceMeta, subs map[string]interface{}) error
	GetResource(url string) (*v1.ResourceInstance, error)
}

type agentResourceManager struct {
	agentResource              *apiv1.ResourceInstance
	prevAgentResHash           uint64
	apicClient                 executeAPIClient
	cfg                        config.CentralConfig
	agentResourceChangeHandler func()
	agentDetails               map[string]interface{}
}

// NewAgentResourceManager - Create a new agent resource manager
func NewAgentResourceManager(cfg config.CentralConfig, apicClient executeAPIClient, agentResourceChangeHandler func()) (Manager, error) {
	m := &agentResourceManager{
		cfg:                        cfg,
		apicClient:                 apicClient,
		agentResourceChangeHandler: agentResourceChangeHandler,
		agentDetails:               make(map[string]interface{}),
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

// FetchAgentResource - Gets the agent resource using API call to apiserver
func (a *agentResourceManager) FetchAgentResource() error {
	if a.cfg.GetAgentName() == "" {
		return nil
	}

	var err error
	a.agentResource, err = a.getAgentResource()
	if err != nil {
		return err
	}

	a.onResourceChange()
	return nil
}

// UpdateAgentStatus - Updates the agent status in agent resource
func (a *agentResourceManager) UpdateAgentStatus(status, prevStatus, message string) error {
	if a.cfg == nil || a.cfg.GetAgentName() == "" {
		return nil
	}

	if a.agentResource == nil {
		return nil
	}

	agentInstance := a.getAgentResourceType()
	// using discovery agent status here, but all agent status resources have the same structure
	agentInstance.SubResources["status"] = management.DiscoveryAgentStatus{
		Version:                config.AgentVersion,
		LatestAvailableVersion: config.AgentLatestVersion,
		State:                  status,
		PreviousState:          prevStatus,
		Message:                message,
		LastActivityTime:       getTimestamp(),
		SdkVersion:             config.SDKVersion,
	}

	// add any details
	if len(a.agentDetails) > 0 {
		util.SetAgentDetails(agentInstance, a.agentDetails)
	}

	err := a.apicClient.CreateSubResource(agentInstance.ResourceMeta, agentInstance.SubResources)
	return err
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

func applyResConfigToCentralConfig(cfg *config.CentralConfiguration, resCfgAdditionalTags, resCfgTeamName, resCfgLogLevel string) {
	agentResLogLevel := log.GlobalLoggerConfig.GetLevel()
	if resCfgLogLevel != "" && !strings.EqualFold(agentResLogLevel, resCfgLogLevel) {
		log.GlobalLoggerConfig.Level(resCfgLogLevel).Apply()
	}

	if resCfgAdditionalTags != "" && !strings.EqualFold(cfg.TagsToPublish, resCfgAdditionalTags) {
		cfg.TagsToPublish = resCfgAdditionalTags
	}

	// If config team is blank, check resource team name.  If resource team name is not blank, use resource team name
	if resCfgTeamName != "" && !strings.EqualFold(cfg.TeamName, resCfgTeamName) {
		cfg.TeamName = resCfgTeamName
	}
}

func (a *agentResourceManager) onResourceChange() {
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
		agentRes = management.NewDiscoveryAgent(a.cfg.GetAgentName(), a.cfg.GetEnvironmentName())
	case config.TraceabilityAgent:
		agentRes = management.NewTraceabilityAgent(a.cfg.GetAgentName(), a.cfg.GetEnvironmentName())
	case config.GovernanceAgent:
		agentRes = management.NewGovernanceAgent(a.cfg.GetAgentName(), a.cfg.GetEnvironmentName())
	}
	var agentInstance *v1.ResourceInstance
	if agentRes != nil {
		agentInstance, _ = agentRes.AsInstance()
	}
	return agentInstance
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

	switch a.getAgentResourceType() {
	case management.DiscoveryAgentGVK().Kind:
		mergeDiscoveryAgentWithConfig(a.GetAgentResource(), a.cfg.(*config.CentralConfiguration))
	case management.TraceabilityAgentGVK().Kind
		mergeTraceabilityAgentWithConfig(a.GetAgentResource(), a.cfg.(*config.CentralConfiguration))
	case management.GovernanceAgentGVK().Kind:
		mergeGovernanceAgentWithConfig(a.GetAgentResource(), a.cfg.(*config.CentralConfiguration))
	default:
		panic(ErrUnsupportedAgentType)
	}
}
