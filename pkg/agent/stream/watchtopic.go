package stream

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// getOrCreateWatchTopic attempts to retrieve a watch topic from central, or create one if it does not exist.
func getOrCreateWatchTopic(name, scope string, rc ResourceClient, agentType config.AgentType) (*mv1.WatchTopic, error) {
	ri, err := rc.Get(fmt.Sprintf("/management/v1alpha1/watchtopics/%s", name))

	if err == nil {
		wt := &mv1.WatchTopic{}
		err = wt.FromInstance(ri)
		return wt, err
	}

	var tmplFunc func() string
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

	bts, err := parseWatchTopicTemplate(name, scope, agentResourceKind, tmplFunc)
	if err != nil {
		return nil, err
	}

	return createWatchTopic(bts, rc)
}

// parseWatchTopicTemplate parses a WatchTopic from a template
func parseWatchTopicTemplate(name, scope, agentResourceKind string, tmplFunc func() string) ([]byte, error) {
	tmpl, err := template.New("watch-topic-tmpl").Parse(tmplFunc())
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, WatchTopicValues{
		Name:              name,
		Title:             name,
		Scope:             scope,
		AgentResourceKind: agentResourceKind,
	})

	return buf.Bytes(), err
}

// createWatchTopic creates a WatchTopic
func createWatchTopic(bts []byte, rc ResourceClient) (*mv1.WatchTopic, error) {
	ri, err := rc.Create("/management/v1alpha1/watchtopics", bts)
	if err != nil {
		return nil, err
	}

	wt := &mv1.WatchTopic{}
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

// WatchTopicValues values to populate the watch topic template
type WatchTopicValues struct {
	Name              string
	Title             string
	Scope             string
	AgentResourceKind string
}

// NewDiscoveryWatchTopic creates a WatchTopic template string.
// Using a template instead of unmarshalling into a struct to avoid sending a request with empty fields
func NewDiscoveryWatchTopic() string {
	return `
{
  "group": "management",
  "apiVersion": "v1alpha1",
  "kind": "WatchTopic",
  "name": "{{.Name}}",
  "title": "{{.Title}}",
  "spec": {
    "filters": [
      {
        "group": "management",
        "kind": "{{.AgentResourceKind}}",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "updated"
        ]
      },
      {
        "group": "management",
        "kind": "AccessRequest",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      },
      {
        "group": "management",
        "kind": "APIService",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      },
      {
        "group": "management",
        "kind": "APIServiceInstance",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      },
      {
        "group": "catalog",
        "kind": "Category",
        "name": "*",
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      }
    ],
    "description": "Watch Topic used by a discovery agent for resources in the {{.Scope}} environment."
  }
}`
}

// NewTraceWatchTopic creates a WatchTopic template string
func NewTraceWatchTopic() string {
	return `
{
  "group": "management",
  "apiVersion": "v1alpha1",
  "kind": "WatchTopic",
  "name": "{{.Name}}",
  "title": "{{.Title}}",
  "spec": {
    "filters": [
      {
        "group": "management",
        "kind": "{{.AgentResourceKind}}",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "updated"
        ]
      },
      {
        "group": "management",
        "kind": "AccessRequest",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      },
      {
        "group": "management",
        "kind": "APIService",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      },
      {
        "group": "management",
        "kind": "APIServiceInstance",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      }
    ],
    "description": "Watch Topic used by a traceability agent for resources in the {{.Scope}} environment."
  }
}`
}

// NewGovernanceAgentWatchTopic creates a WatchTopic template string
func NewGovernanceAgentWatchTopic() string {
	return `
{
  "group": "management",
  "apiVersion": "v1alpha1",
  "kind": "WatchTopic",
  "name": "{{.Name}}",
  "title": "{{.Title}}",
  "spec": {
    "filters": [
      {
        "group": "management",
        "kind": "{{.AgentResourceKind}}",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "updated"
        ]
      },
      {
        "group": "management",
        "kind": "AmplifyRuntimeConfig",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      },
      {
        "group": "management",
        "kind": "AccessRequest",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      },
      {
        "group": "management",
        "kind": "APIService",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      },
      {
        "group": "management",
        "kind": "APIServiceInstance",
        "name": "*",
        "scope": {
          "kind": "Environment",
          "name": "{{.Scope}}"
        },
        "type": [
          "created",
          "updated",
          "deleted"
        ]
      }
    ],
    "description": "Watch Topic used by a governance agent for resources in the {{.Scope}} environment."
  }
}`
}
