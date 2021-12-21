package resource

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// AgentTypesMap - Agent Types map
var AgentTypesMap = map[config.AgentType]string{
	config.DiscoveryAgent:    "discoveryagents",
	config.TraceabilityAgent: "traceabilityagents",
	config.GovernanceAgent:   "governanceagents",
}

// Manager - interface to manage agent resource
type Manager interface {
	OnConfigChange(cfg config.CentralConfig, apicClient apic.Client)

	GetAgentResource() *v1.ResourceInstance
	SetAgentResource(agentResource *v1.ResourceInstance)
	FetchAgentResource() error
	UpdateAgentStatus(status, prevStatus, message string) error
}

type agentResourceManager struct {
	agentResource              *v1.ResourceInstance
	prevAgentResHash           uint64
	apicClient                 apic.Client
	cfg                        config.CentralConfig
	agentResourceChangeHandler func()
}

// NewAgentResourceManager - Create a new agent resource manager
func NewAgentResourceManager(cfg config.CentralConfig, apicClient apic.Client, agentResourceChangeHandler func()) (Manager, error) {
	m := &agentResourceManager{
		cfg:                        cfg,
		apicClient:                 apicClient,
		agentResourceChangeHandler: agentResourceChangeHandler,
	}

	if m.getAgentResourceType() != "" {
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
func (a *agentResourceManager) GetAgentResource() *v1.ResourceInstance {
	return a.agentResource
}

// SetAgentResource - Sets the agent resource which triggers agent resource change handler
func (a *agentResourceManager) SetAgentResource(agentResource *v1.ResourceInstance) {
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

	if a.agentResource != nil {
		agentResourceType := a.getAgentResourceType()
		resource, err := a.createAgentStatusSubResource(agentResourceType, status, prevStatus, message)
		if err != nil {
			return err
		}

		err = a.updateAgentStatusAPI(resource, agentResourceType)
		if err != nil {
			return err
		}
	}
	return nil
}

// getTimestamp - Returns current timestamp formatted for API Server
func getTimestamp() v1.Time {
	activityTime := time.Now()
	newV1Time := v1.Time(activityTime)
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
	if a.prevAgentResHash != 0 {
		if a.prevAgentResHash == agentResHash {
			isChanged = false
		}
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

func (a *agentResourceManager) getAgentResourceType() string {
	agentType, ok := AgentTypesMap[a.cfg.GetAgentType()]
	if ok {
		return agentType
	}
	return ""
}

// GetAgentResource - returns the agent resource
func (a *agentResourceManager) getAgentResource() (*v1.ResourceInstance, error) {
	agentResourceType := a.getAgentResourceType()
	agentResourceURL := a.cfg.GetEnvironmentURL() + "/" + agentResourceType + "/" + a.cfg.GetAgentName()

	response, err := a.apicClient.ExecuteAPI(api.GET, agentResourceURL, nil, nil)
	if err != nil {
		return nil, err
	}

	agent := &v1.ResourceInstance{}
	err = json.Unmarshal(response, agent)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (a *agentResourceManager) updateAgentStatusAPI(resource interface{}, agentResourceType string) error {
	buffer, err := json.Marshal(resource)
	if err != nil {
		return nil
	}

	subResURL := a.cfg.GetEnvironmentURL() + "/" + agentResourceType + "/" + a.cfg.GetAgentName() + "/status"
	_, err = a.apicClient.ExecuteAPI(api.PUT, subResURL, nil, buffer)
	if err != nil {
		return err
	}
	return nil
}

func (a *agentResourceManager) createAgentStatusSubResource(agentResourceType, status, prevStatus, message string) (*v1.ResourceInstance, error) {
	switch agentResourceType {
	case v1alpha1.DiscoveryAgentResourceName:
		agentRes := createDiscoveryAgentStatusResource(a.cfg.GetAgentName(), status, prevStatus, message)
		resourceInstance, _ := agentRes.AsInstance()
		return resourceInstance, nil
	case v1alpha1.TraceabilityAgentResourceName:
		agentRes := createTraceabilityAgentStatusResource(a.cfg.GetAgentName(), status, prevStatus, message)
		resourceInstance, _ := agentRes.AsInstance()
		return resourceInstance, nil
	case v1alpha1.GovernanceAgentResourceName:
		agentRes := createGovernanceAgentStatusResource(a.cfg.GetAgentName(), status, prevStatus, message)
		resourceInstance, _ := agentRes.AsInstance()
		return resourceInstance, nil
	default:
		return nil, ErrUnsupportedAgentType
	}
}

func (a *agentResourceManager) mergeResourceWithConfig() {
	// IMP - To be removed once the model is in production
	if a.cfg.GetAgentName() == "" {
		return
	}

	switch a.getAgentResourceType() {
	case v1alpha1.DiscoveryAgentResourceName:
		mergeDiscoveryAgentWithConfig(a.GetAgentResource(), a.cfg.(*config.CentralConfiguration))
	case v1alpha1.TraceabilityAgentResourceName:
		mergeTraceabilityAgentWithConfig(a.GetAgentResource(), a.cfg.(*config.CentralConfiguration))
	case v1alpha1.GovernanceAgentResourceName:
		mergeGovernanceAgentWithConfig(a.GetAgentResource(), a.cfg.(*config.CentralConfiguration))
	default:
		panic(ErrUnsupportedAgentType)
	}
}
