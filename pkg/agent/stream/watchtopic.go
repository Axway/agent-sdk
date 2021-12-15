package stream

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/errors"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// GetOrCreateWatchTopic attempts to retrieve a watch topic from central, or create one if it does not exist.
func GetOrCreateWatchTopic(name, scope string, rc ResourceClient, agentType config.AgentType) (*mv1.WatchTopic, error) {
	ri, err := rc.Get(fmt.Sprintf("/management/v1alpha1/watchtopics/%s", name))

	if err == nil {
		wt := &mv1.WatchTopic{}
		err = wt.FromInstance(ri)
		return wt, err
	}

	var tmplFunc func() string
	switch agentType {
	case config.DiscoveryAgent:
		tmplFunc = NewDiscoveryWatchTopic
	case config.TraceabilityAgent:
		tmplFunc = NewTraceWatchTopic
	case config.GovernanceAgent:
		// TODO
	default:
		return nil, errors.New(1000, "unsupported agent type")
	}

	bts, err := parseWatchTopicTemplate(name, scope, tmplFunc)
	if err != nil {
		return nil, err
	}

	return CreateWatchTopic(bts, rc)
}

// parseWatchTopicTemplate parses a WatchTopic from a template
func parseWatchTopicTemplate(name, scope string, tmplFunc func() string) ([]byte, error) {
	tmpl, err := template.New("watch-topic-tmpl").Parse(tmplFunc())
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, WatchTopicValues{
		Name:  name,
		Title: name,
		Scope: scope,
	})

	return buf.Bytes(), err
}

// CreateWatchTopic creates a WatchTopic
func CreateWatchTopic(bts []byte, rc ResourceClient) (*mv1.WatchTopic, error) {
	ri, err := rc.Create("/management/v1alpha1/watchtopics", bts)
	if err != nil {
		return nil, err
	}

	wt := &mv1.WatchTopic{}
	err = wt.FromInstance(ri)

	return wt, err
}

// GetCachedWatchTopic checks the cache for a saved WatchTopic ResourceClient
func GetCachedWatchTopic(c cache.GetItem, key string) (*mv1.WatchTopic, error) {
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
	Name  string
	Title string
	Scope string
}

// NewDiscoveryWatchTopic creates a WatchTopic template string
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
