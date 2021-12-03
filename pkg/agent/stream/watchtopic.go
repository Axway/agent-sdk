package stream

import (
	"bytes"
	"fmt"
	"text/template"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
)

// WatchTopicName naming pattern for creating watch topics
func WatchTopicName(env, agent string) string {
	return fmt.Sprintf("%s-%s", env, agent)
}

func GetOrCreateWatchTopic(name, scope string, rc ResourceClient) (*mv1.WatchTopic, error) {
	ri, err := rc.Get(fmt.Sprintf("/management/v1alpha1/watchtopics/%s", name))
	if err != nil {
		return CreateWatchTopic(name, scope, rc)
	}

	wt := &mv1.WatchTopic{}
	err = wt.FromInstance(ri)

	return wt, err
}

func CreateWatchTopic(name, scope string, rc ResourceClient) (*mv1.WatchTopic, error) {
	tmplString := NewWatchTopic()
	tmpl, err := template.New("watch-topic-tmpl").Parse(tmplString)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, WatchTopicValues{
		Name:  name,
		Title: name,
		Scope: scope,
	})
	if err != nil {
		return nil, err
	}

	ri, err := rc.Create("/management/v1alpha1/watchtopics", buf.Bytes())
	if err != nil {
		return nil, err
	}

	wt := &mv1.WatchTopic{}
	err = wt.FromInstance(ri)

	return wt, err
}

// GetCachedWatchTopic checks the cache for a saved WatchTopic ResourceClient
func GetCachedWatchTopic(c cache.Cache, key string) (*mv1.WatchTopic, error) {
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

type WatchTopicValues struct {
	Name  string
	Title string
	Scope string
}

// NewWatchTopic creates a WatchTopic ResourceClient
func NewWatchTopic() string {
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
    "description": "Watch Topic for resources in the {{.Scope}} environment."
  }
}`
}
