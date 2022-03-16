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
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

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
				"group": "{{if .Group}}{{.Group}}{{else}}management{{end}}",
				"kind": "{{.KindName}}",
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
func getOrCreateWatchTopic(name, scope string, client apiClient, agentType config.AgentType) (*mv1.WatchTopic, error) {
	wt := emptyWatchTopic()
	ri, err := client.GetResource(fmt.Sprintf("%s/%s", wt.GetKindLink(), name))

	if err == nil {
		err = wt.FromInstance(ri)
		return wt, err
	}

	var tmplValuesFunc func(string, string, string) WatchTopicValues
	agentResourceKind := ""
	switch agentType {
	case config.DiscoveryAgent:
		agentResourceKind = "DiscoveryAgent"
		tmplValuesFunc = NewDiscoveryWatchTopic
	case config.TraceabilityAgent:
		agentResourceKind = "TraceabilityAgent"
		tmplValuesFunc = NewTraceWatchTopic
	case config.GovernanceAgent:
		agentResourceKind = "GovernanceAgent"
		tmplValuesFunc = NewGovernanceAgentWatchTopic
	default:
		return nil, resource.ErrUnsupportedAgentType
	}

	bts, err := parseWatchTopicTemplate(tmplValuesFunc(name, scope, agentResourceKind))
	if err != nil {
		return nil, err
	}

	return createWatchTopic(bts, client)
}

func emptyWatchTopic() *mv1.WatchTopic {
	return &mv1.WatchTopic{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1.WatchTopicGVK(),
		},
	}
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
	wt := emptyWatchTopic()
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
	KindName   string
	EventTypes []string
	Group      string // blank defaults to management in template
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
func NewDiscoveryWatchTopic(name, scope, agentResourceKind string) WatchTopicValues {
	return WatchTopicValues{
		Name:        name,
		Title:       name,
		Description: fmt.Sprintf(desc, "discovery", scope),
		Kinds: []kindValues{
			{KindName: agentResourceKind, ScopeName: scope, EventTypes: updated},
			{KindName: "Category", Group: "category", EventTypes: all},
			{KindName: "APIService", ScopeName: scope, EventTypes: all},
			{KindName: "APIServiceInstance", ScopeName: scope, EventTypes: all},
			{KindName: "Credential", ScopeName: scope, EventTypes: createdOrUpdated},
			{KindName: "AccessRequest", ScopeName: scope, EventTypes: createdOrUpdated},
			{KindName: "ManagedApplication", ScopeName: scope, EventTypes: createdOrUpdated},
			{KindName: "CredentialRequestDefinition", ScopeName: scope, EventTypes: all},
			{KindName: "AccessRequestDefinition", ScopeName: scope, EventTypes: all},
		},
	}
}

// NewTraceWatchTopic creates a WatchTopic template string
func NewTraceWatchTopic(name, scope, agentResourceKind string) WatchTopicValues {
	return WatchTopicValues{
		Name:        name,
		Title:       name,
		Description: fmt.Sprintf(desc, "traceability", scope),
		Kinds: []kindValues{
			{KindName: agentResourceKind, ScopeName: scope, EventTypes: updated},
			{KindName: "APIService", ScopeName: scope, EventTypes: all},
			{KindName: "APIServiceInstance", ScopeName: scope, EventTypes: all},
			{KindName: "AccessRequest", ScopeName: scope, EventTypes: all},
			{KindName: "ManagedApplication", ScopeName: scope, EventTypes: createdOrUpdated},
		},
	}
}

// NewGovernanceAgentWatchTopic creates a WatchTopic template string
func NewGovernanceAgentWatchTopic(name, scope, agentResourceKind string) WatchTopicValues {
	return WatchTopicValues{
		Name:        name,
		Title:       name,
		Description: fmt.Sprintf(desc, "governance", scope),
		Kinds: []kindValues{
			{KindName: agentResourceKind, ScopeName: scope, EventTypes: updated},
			{KindName: "AmplifyRuntimeConfig", ScopeName: scope, EventTypes: all},
			{KindName: "AccessRequest", ScopeName: scope, EventTypes: all},
			{KindName: "APIService", ScopeName: scope, EventTypes: all},
			{KindName: "APIServiceInstance", ScopeName: scope, EventTypes: all},
			{KindName: "ManagedApplication", ScopeName: scope, EventTypes: createdOrUpdated},
		},
	}
}
