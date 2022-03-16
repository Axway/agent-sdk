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
		"filters": [
			{
				"group": "catalog",
				"kind": "Category",
				"name": "*",
				"type": ["created", "updated", "deleted"]
			},{{range $index, $kind := .Kinds}}{{if $index}},{{end}}
			{
				"group": "management",
				"kind": "{{.KindName}}",
				"name": "*",
				"scope": {
					"kind": "Environment",
					"name": "{{.Scope}}"
				},
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

	var tmplFunc func(string, string, string) ([]byte, error)
	agentResourceKind := ""
	switch agentType {
	case config.DiscoveryAgent:
		agentResourceKind = "DiscoveryAgent"
		tmplFunc = NewDiscoveryWatchTopic
	case config.TraceabilityAgent:
		agentResourceKind = "TraceabilityAgent"
		tmplFunc = NewTraceWatchTopic
	case config.GovernanceAgent:
		agentResourceKind = "GovernanceAgent"
		tmplFunc = NewGovernanceAgentWatchTopic
	default:
		return nil, resource.ErrUnsupportedAgentType
	}

	bts, err := tmplFunc(name, scope, agentResourceKind)
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
func executeTemplate(values WatchTopicValues) ([]byte, error) {
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
	Scope      string
}

// WatchTopicValues values to populate the watch topic template
type WatchTopicValues struct {
	Name              string
	Title             string
	AgentResourceKind string
	Description       string
	Kinds             []kindValues
}

// NewDiscoveryWatchTopic creates a WatchTopic template string.
// Using a template instead of unmarshalling into a struct to avoid sending a request with empty fields
func NewDiscoveryWatchTopic(name, scope, agentResourceKind string) ([]byte, error) {
	wtValues := WatchTopicValues{
		Name:              name,
		Title:             name,
		Description:       fmt.Sprintf(desc, "discovery", scope),
		AgentResourceKind: agentResourceKind,
		Kinds: []kindValues{
			{KindName: agentResourceKind, Scope: scope, EventTypes: updated},
			{KindName: "APIService", Scope: scope, EventTypes: all},
			{KindName: "APIServiceInstance", Scope: scope, EventTypes: all},
			{KindName: "Credential", Scope: scope, EventTypes: createdOrUpdated},
			{KindName: "AccessRequest", Scope: scope, EventTypes: createdOrUpdated},
			{KindName: "ManagedApplication", Scope: scope, EventTypes: createdOrUpdated},
			{KindName: "CredentialRequestDefinition", Scope: scope, EventTypes: all},
			{KindName: "AccessRequestDefinition", Scope: scope, EventTypes: all},
		},
	}

	return executeTemplate(wtValues)
}

// NewTraceWatchTopic creates a WatchTopic template string
func NewTraceWatchTopic(name, scope, agentResourceKind string) ([]byte, error) {
	wtValues := WatchTopicValues{
		Name:              name,
		Title:             name,
		Description:       fmt.Sprintf(desc, "traceability", scope),
		AgentResourceKind: agentResourceKind,
		Kinds: []kindValues{
			{KindName: agentResourceKind, Scope: scope, EventTypes: updated},
			{KindName: "APIService", Scope: scope, EventTypes: all},
			{KindName: "APIServiceInstance", Scope: scope, EventTypes: all},
			{KindName: "AccessRequest", Scope: scope, EventTypes: all},
			{KindName: "ManagedApplication", Scope: scope, EventTypes: createdOrUpdated},
		},
	}

	return executeTemplate(wtValues)
}

// NewGovernanceAgentWatchTopic creates a WatchTopic template string
func NewGovernanceAgentWatchTopic(name, scope, agentResourceKind string) ([]byte, error) {
	wtValues := WatchTopicValues{
		Name:              name,
		Title:             name,
		Description:       fmt.Sprintf(desc, "governance", scope),
		AgentResourceKind: agentResourceKind,
		Kinds: []kindValues{
			{KindName: agentResourceKind, Scope: scope, EventTypes: updated},
			{KindName: "AmplifyRuntimeConfig", Scope: scope, EventTypes: all},
			{KindName: "AccessRequest", Scope: scope, EventTypes: all},
			{KindName: "APIService", Scope: scope, EventTypes: all},
			{KindName: "APIServiceInstance", Scope: scope, EventTypes: all},
			{KindName: "ManagedApplication", Scope: scope, EventTypes: createdOrUpdated},
		},
	}

	return executeTemplate(wtValues)
}
