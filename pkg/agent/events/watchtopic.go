package events

import (
	"bytes"
	_ "embed" // load of the watch topic template
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/config"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

//go:embed assets/watch-topic-template.json
var agentTemplate string

var agentTypesMap = map[config.AgentType]string{
	config.DiscoveryAgent:    management.DiscoveryAgentResourceName,
	config.TraceabilityAgent: management.TraceabilityAgentResourceName,
	config.ComplianceAgent:   management.ComplianceAgentResourceName,
}

type watchTopicFeatures interface {
	GetAgentType() config.AgentType
	GetWatchResourceFilters() []config.ResourceFilter
}

const (
	desc = "Watch Topic used by a %s agent for resources in the %s environment."
	// WatchTopicFilterTypeCreated filter type name
	WatchTopicFilterTypeCreated = "created"
	// WatchTopicFilterTypeUpdated filter type name
	WatchTopicFilterTypeUpdated = "updated"
	// WatchTopicFilterTypeDeleted filter type name
	WatchTopicFilterTypeDeleted = "deleted"
)

var (
	created          = []string{WatchTopicFilterTypeCreated}
	updated          = []string{WatchTopicFilterTypeUpdated}
	deleted          = []string{WatchTopicFilterTypeDeleted}
	createdOrUpdated = append(created, updated...)
	all              = append(createdOrUpdated, deleted...)
)

// getOrCreateWatchTopic attempts to retrieve a watch topic from central, or create one if it does not exist.
func getOrCreateWatchTopic(name, scope string, client APIClient, features watchTopicFeatures) (*management.WatchTopic, error) {
	wt := management.NewWatchTopic("")
	ri, err := client.GetResource(fmt.Sprintf("%s/%s", wt.GetKindLink(), name))

	if err == nil {
		err = wt.FromInstance(ri)
		if err != nil {
			return nil, err
		}
	}

	var agentResourceGroupKind v1.GroupKind
	var tmplValuesFunc func(string, string, v1.GroupKind, watchTopicFeatures) WatchTopicValues

	switch features.GetAgentType() {
	case config.DiscoveryAgent:
		agentResourceGroupKind = management.DiscoveryAgentGVK().GroupKind
		tmplValuesFunc = NewDiscoveryWatchTopic
	case config.TraceabilityAgent:
		agentResourceGroupKind = management.TraceabilityAgentGVK().GroupKind
		tmplValuesFunc = NewTraceWatchTopic
	case config.ComplianceAgent:
		agentResourceGroupKind = management.ComplianceAgentGVK().GroupKind
		tmplValuesFunc = NewComplianceWatchTopic
	default:
		return nil, resource.ErrUnsupportedAgentType
	}

	newWT, err := parseWatchTopicTemplate(tmplValuesFunc(name, scope, agentResourceGroupKind, features))
	if err != nil {
		return nil, err
	}

	filters := features.GetWatchResourceFilters()
	for _, filter := range filters {
		eventTypes := make([]string, 0)
		for _, filterEventType := range filter.EventTypes {
			eventTypes = append(eventTypes, (string(filterEventType)))
		}

		wf := management.WatchTopicSpecFilters{
			Group: filter.Group,
			Kind:  filter.Kind,
			Name:  filter.Name,
			Type:  eventTypes,
		}

		if filter.Scope != nil {
			wf.Scope = &management.WatchTopicSpecScope{
				Kind: filter.Scope.Kind,
				Name: filter.Scope.Name,
			}
		}

		newWT.Spec.Filters = append(newWT.Spec.Filters, wf)
	}

	// if the existing wt has no name then it does not exist yet
	if wt.Name == "" {
		return createOrUpdateWatchTopic(newWT, client)
	}

	// compare the generated WT and the existing WT for changes
	if shouldPushUpdate(wt, newWT) {
		// update the spec in the existing watch topic
		wt.Spec = newWT.Spec
		return createOrUpdateWatchTopic(wt, client)
	}

	return wt, nil
}

func shouldPushUpdate(cur, new *management.WatchTopic) bool {
	filtersDiff := func(a, b []management.WatchTopicSpecFilters) bool {
		for _, aFilter := range a {
			found := false
			for _, bFilter := range b {
				if filtersEqual(aFilter, bFilter) {
					found = true
					break
				}
			}
			if !found {
				// update required
				return true
			}
		}
		return false
	}

	if filtersDiff(cur.Spec.Filters, new.Spec.Filters) {
		return true
	}
	return filtersDiff(new.Spec.Filters, cur.Spec.Filters)
}

func filtersEqual(a, b management.WatchTopicSpecFilters) (equal bool) {
	if a.Group != b.Group ||
		a.Kind != b.Kind ||
		a.Name != b.Name ||
		a.Scope == nil && b.Scope != nil ||
		a.Scope != nil && b.Scope == nil {
		return
	}

	if a.Scope != nil && b.Scope != nil {
		if a.Scope.Kind != b.Scope.Kind ||
			a.Scope.Name != b.Scope.Name {
			return
		}
	}

	if areTypesEqual(a.Type, b.Type) {
		return false
	}
	return !areTypesEqual(b.Type, a.Type)
}

func areTypesEqual(aTypes, bTypes []string) bool {
	for _, aType := range aTypes {
		found := false
		for _, bType := range bTypes {
			if aType == bType {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}
	return false
}

// executeTemplate parses a WatchTopic from a template
func parseWatchTopicTemplate(values WatchTopicValues) (*management.WatchTopic, error) {
	tmpl, err := template.New("watch-topic-tmpl").Funcs(template.FuncMap{"StringsJoin": strings.Join}).Parse(agentTemplate)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, values)
	if err != nil {
		return nil, err
	}

	wt := management.NewWatchTopic("")
	err = json.Unmarshal(buf.Bytes(), wt)
	return wt, err
}

// createOrUpdateWatchTopic creates a WatchTopic
func createOrUpdateWatchTopic(wt *management.WatchTopic, rc APIClient) (*management.WatchTopic, error) {
	if wt.Metadata.ID != "" {
		err := rc.DeleteResourceInstance(wt)
		if err != nil {
			return nil, err
		}
	}

	ri, err := rc.CreateResourceInstance(wt)
	if err != nil {
		return nil, err
	}

	err = wt.FromInstance(ri)

	return wt, err
}

type kindValues struct {
	v1.GroupKind
	EventTypes []string
	ScopeKind  string // blank defaults to Environment in template
	ScopeName  string // blank generates no scope in template
	Name       string
}

// WatchTopicValues values to populate the watch topic template
type WatchTopicValues struct {
	Name        string
	Title       string
	Description string
	Kinds       []kindValues
}

// NewDiscoveryWatchTopic creates a WatchTopic template string.
// Using a template instead of unmarshalling into a struct to avoid sending a request with empty fields
func NewDiscoveryWatchTopic(name, scope string, agentResourceGroupKind v1.GroupKind, features watchTopicFeatures) WatchTopicValues {
	kinds := []kindValues{
		{GroupKind: agentResourceGroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: updated},
		{GroupKind: management.APIServiceGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.APIServiceInstanceGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.AccessControlListGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.CredentialGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: createdOrUpdated},
		{GroupKind: management.AccessRequestGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: createdOrUpdated},
		{GroupKind: management.ManagedApplicationGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: createdOrUpdated},
		{GroupKind: management.CredentialRequestDefinitionGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.AccessRequestDefinitionGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.ManagedApplicationProfileGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: createdOrUpdated},
		{GroupKind: management.ApplicationProfileDefinitionGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.EnvironmentGVK().GroupKind, Name: scope, EventTypes: updated},
	}

	return WatchTopicValues{
		Name:        name,
		Title:       name,
		Description: fmt.Sprintf(desc, "discovery", scope),
		Kinds:       kinds,
	}
}

// NewTraceWatchTopic creates a WatchTopic template string
func NewTraceWatchTopic(name, scope string, agentResourceGroupKind v1.GroupKind, features watchTopicFeatures) WatchTopicValues {
	kinds := []kindValues{
		{GroupKind: agentResourceGroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: updated},
		{GroupKind: management.APIServiceGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.APIServiceInstanceGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.AccessRequestGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.ManagedApplicationGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
	}

	return WatchTopicValues{
		Name:        name,
		Title:       name,
		Description: fmt.Sprintf(desc, "traceability", scope),
		Kinds:       kinds,
	}
}

// NewComplianceWatchTopic creates a WatchTopic template string
func NewComplianceWatchTopic(name, scope string, agentResourceGroupKind v1.GroupKind, features watchTopicFeatures) WatchTopicValues {
	kinds := []kindValues{
		{GroupKind: agentResourceGroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: updated},
		{GroupKind: management.APIServiceGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
		{GroupKind: management.APIServiceInstanceGVK().GroupKind, ScopeName: scope, ScopeKind: management.EnvironmentGVK().Kind, EventTypes: all},
	}

	return WatchTopicValues{
		Name:        name,
		Title:       name,
		Description: fmt.Sprintf(desc, "compliance", scope),
		Kinds:       kinds,
	}
}

// GetWatchTopic retrieves a watch topic based on the agent config. Creates a watch topic if one does not exist.
func GetWatchTopic(cfg config.CentralConfig, client APIClient) (*management.WatchTopic, error) {
	env := cfg.GetEnvironmentName()

	wtName := getWatchTopicName(env, cfg.GetAgentType())
	wt, err := getOrCreateWatchTopic(wtName, env, client, cfg)
	if err != nil {
		return nil, err
	}

	return wt, err
}

func getWatchTopicName(envName string, agentType config.AgentType) string {
	return envName + getWatchTopicNameSuffix(agentType)
}

func getWatchTopicNameSuffix(agentType config.AgentType) string {
	return "-" + agentTypesMap[agentType]
}
