package stream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	cv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

type watchTopicFeatures interface {
	IsMarketplaceSubsEnabled() bool
	GetAgentType() config.AgentType
}

const (
	agentTemplate = `{
	"group": "management",
	"apiVersion": "v1alpha1",
	"kind": "WatchTopic",
	"name": "{{.Name}}",
	"title": "{{.Title}}",
	"spec": {
		"filters": [{{range $index, $kind := .Kinds}}{{if $index}},{{end}}
			{
				"group": "{{.Group}}",
				"kind": "{{.Kind}}",
				"name": "*",
				{{if ne .ScopeName ""}}"scope": {
					"kind": "{{if .ScopeKind}}{{.ScopeKind}}{{else}}Environment{{end}}",
					"name": "{{.ScopeName}}"
				},{{end}}
				"type": ["{{ StringsJoin .EventTypes "\",\""}}"]
			}{{end}}
		],
		"description": "{{.Description}}"
	}
}
`
	desc = "Watch Topic used by a %s agent for resources in the %s environment."
)

var (
	created          = []string{"created"}
	updated          = []string{"updated"}
	deleted          = []string{"deleted"}
	createdOrUpdated = append(created, updated...)
	all              = append(createdOrUpdated, deleted...)
)

// getOrCreateWatchTopic attempts to retrieve a watch topic from central, or create one if it does not exist.
func getOrCreateWatchTopic(name, scope string, client apiClient, features watchTopicFeatures) (*mv1.WatchTopic, error) {
	wt := mv1.NewWatchTopic("")
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
		agentResourceGroupKind = mv1.DiscoveryAgentGVK().GroupKind
		tmplValuesFunc = NewDiscoveryWatchTopic
	case config.TraceabilityAgent:
		agentResourceGroupKind = mv1.TraceabilityAgentGVK().GroupKind
		tmplValuesFunc = NewTraceWatchTopic
	case config.GovernanceAgent:
		agentResourceGroupKind = mv1.GovernanceAgentGVK().GroupKind
		tmplValuesFunc = NewGovernanceAgentWatchTopic
	default:
		return nil, resource.ErrUnsupportedAgentType
	}

	newWT, err := parseWatchTopicTemplate(tmplValuesFunc(name, scope, agentResourceGroupKind, features))
	if err != nil {
		return nil, err
	}

	// if the existing wt has no name then it does not exist yet
	if wt.Name == "" {
		return createOrUpdateWatchTopic(newWT, client)
	}

	//compare the generated WT and the existing WT for changes
	if shouldPushUpdate(wt, newWT) {
		// update the spec in the existing watch topic
		wt.Spec = newWT.Spec
		return createOrUpdateWatchTopic(wt, client)
	}

	return wt, nil
}

func shouldPushUpdate(cur, new *mv1.WatchTopic) bool {
	for _, newFilter := range new.Spec.Filters {
		found := false
		for _, existingFilter := range cur.Spec.Filters {
			if filtersEqual(newFilter, existingFilter) {
				found = true
				break
			}
		}
		if !found {
			// update required
			return true
		}
	}

	// now check the reverse
	for _, existingFilter := range cur.Spec.Filters {
		found := false
		for _, newFilter := range new.Spec.Filters {
			if filtersEqual(existingFilter, newFilter) {
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

func filtersEqual(a, b mv1.WatchTopicSpecFilters) (equal bool) {
	if a.Group != b.Group {
		return
	}

	if a.Kind != b.Kind {
		return
	}

	if a.Name != b.Name {
		return
	}

	if a.Scope == nil && b.Scope != nil {
		return
	}

	if a.Scope != nil && b.Scope == nil {
		return
	}

	if a.Scope != nil && b.Scope != nil {
		if a.Scope.Kind != b.Scope.Kind {
			return
		}

		if a.Scope.Name != b.Scope.Name {
			return
		}
	}

	for _, aType := range a.Type {
		found := false
		for _, bType := range b.Type {
			if aType == bType {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}

	for _, bType := range b.Type {
		found := false
		for _, aType := range a.Type {
			if aType == bType {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}
	return true
}

// executeTemplate parses a WatchTopic from a template
func parseWatchTopicTemplate(values WatchTopicValues) (*mv1.WatchTopic, error) {
	tmpl, err := template.New("watch-topic-tmpl").Funcs(template.FuncMap{"StringsJoin": strings.Join}).Parse(agentTemplate)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, values)
	if err != nil {
		return nil, err
	}

	wt := mv1.NewWatchTopic("")
	err = json.Unmarshal(buf.Bytes(), wt)
	return wt, err
}

// createOrUpdateWatchTopic creates a WatchTopic
func createOrUpdateWatchTopic(wt *mv1.WatchTopic, rc apiClient) (*mv1.WatchTopic, error) {
	bts, err := json.Marshal(wt)
	if err != nil {
		return nil, err
	}

	var ri *v1.ResourceInstance
	if wt.Metadata.ID != "" {
		ri, err = rc.UpdateResource(wt.GetSelfLink(), bts)
	} else {
		ri, err = rc.CreateResource(wt.GetKindLink(), bts)
	}

	if err != nil {
		return nil, err
	}

	err = wt.FromInstance(ri)

	return wt, err
}

// getCachedWatchTopic checks the cache for a saved WatchTopic ResourceClient
func getCachedWatchTopic(c cache.GetItem, key string) (*mv1.WatchTopic, error) {
	item, err := c.Get(key)
	if err != nil {
		return nil, err
	}

	v, ok := item.(*mv1.WatchTopic)
	if !ok {
		return nil, fmt.Errorf("found item for %s, but it is not a *WatchTopic", key)
	}

	return v, nil
}

type kindValues struct {
	v1.GroupKind
	EventTypes []string
	ScopeKind  string // blank defaults to Environment in template
	ScopeName  string // blank generates no scope in template
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
		{GroupKind: agentResourceGroupKind, ScopeName: scope, EventTypes: updated},
		{GroupKind: cv1.CategoryGVK().GroupKind, EventTypes: all},
		{GroupKind: mv1.APIServiceGVK().GroupKind, ScopeName: scope, EventTypes: all},
		{GroupKind: mv1.APIServiceInstanceGVK().GroupKind, ScopeName: scope, EventTypes: all},
	}
	if features.IsMarketplaceSubsEnabled() {
		kinds = append(kinds, []kindValues{
			{GroupKind: mv1.CredentialGVK().GroupKind, ScopeName: scope, EventTypes: createdOrUpdated},
			{GroupKind: mv1.AccessRequestGVK().GroupKind, ScopeName: scope, EventTypes: createdOrUpdated},
			{GroupKind: mv1.ManagedApplicationGVK().GroupKind, ScopeName: scope, EventTypes: createdOrUpdated},
			{GroupKind: mv1.CredentialRequestDefinitionGVK().GroupKind, ScopeName: scope, EventTypes: all},
			{GroupKind: mv1.AccessRequestDefinitionGVK().GroupKind, ScopeName: scope, EventTypes: all},
		}...)
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
		{GroupKind: agentResourceGroupKind, ScopeName: scope, EventTypes: updated},
		{GroupKind: mv1.APIServiceGVK().GroupKind, ScopeName: scope, EventTypes: all},
		{GroupKind: mv1.APIServiceInstanceGVK().GroupKind, ScopeName: scope, EventTypes: all},
	}
	if features.IsMarketplaceSubsEnabled() {
		kinds = append(kinds, []kindValues{
			{GroupKind: mv1.AccessRequestGVK().GroupKind, ScopeName: scope, EventTypes: all},
			{GroupKind: mv1.ManagedApplicationGVK().GroupKind, ScopeName: scope, EventTypes: all},
		}...)
	}
	return WatchTopicValues{
		Name:        name,
		Title:       name,
		Description: fmt.Sprintf(desc, "traceability", scope),
		Kinds:       kinds,
	}
}

// NewGovernanceAgentWatchTopic creates a WatchTopic template string
func NewGovernanceAgentWatchTopic(name, scope string, agentResourceGroupKind v1.GroupKind, features watchTopicFeatures) WatchTopicValues {
	kinds := []kindValues{
		{GroupKind: agentResourceGroupKind, ScopeName: scope, EventTypes: updated},
		{GroupKind: mv1.AmplifyRuntimeConfigGVK().GroupKind, ScopeName: scope, EventTypes: all},
		{GroupKind: mv1.APIServiceGVK().GroupKind, ScopeName: scope, EventTypes: all},
		{GroupKind: mv1.APIServiceInstanceGVK().GroupKind, ScopeName: scope, EventTypes: all},
	}
	if features.IsMarketplaceSubsEnabled() {
		kinds = append(kinds, []kindValues{
			{GroupKind: mv1.AccessRequestGVK().GroupKind, ScopeName: scope, EventTypes: all},
			{GroupKind: mv1.ManagedApplicationGVK().GroupKind, ScopeName: scope, EventTypes: createdOrUpdated},
		}...)
	}
	return WatchTopicValues{
		Name:        name,
		Title:       name,
		Description: fmt.Sprintf(desc, "governance", scope),
		Kinds:       kinds,
	}
}
