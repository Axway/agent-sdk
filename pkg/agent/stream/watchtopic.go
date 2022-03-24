package stream

import (
	"bytes"
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
				{{if .ScopeName}}"scope": {
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
		return wt, err
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

	bts, err := parseWatchTopicTemplate(tmplValuesFunc(name, scope, agentResourceGroupKind, features))
	if err != nil {
		return nil, err
	}

	return createWatchTopic(bts, client)
}

// executeTemplate parses a WatchTopic from a template
func parseWatchTopicTemplate(values WatchTopicValues) ([]byte, error) {
	tmpl, err := template.New("watch-topic-tmpl").Funcs(template.FuncMap{"StringsJoin": strings.Join}).Parse(agentTemplate)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, values)

	return buf.Bytes(), err
}

// createWatchTopic creates a WatchTopic
func createWatchTopic(bts []byte, rc apiClient) (*mv1.WatchTopic, error) {
	wt := mv1.NewWatchTopic("")
	ri, err := rc.CreateResource(wt.GetKindLink(), bts)
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
