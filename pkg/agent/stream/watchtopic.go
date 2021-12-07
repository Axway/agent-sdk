package stream

import (
	"bytes"
	"fmt"
	"text/template"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
)

// GetOrCreateWatchTopic attempts to retrieve a watch topic from central, or create one if it does not exist.
func GetOrCreateWatchTopic(name, scope string, rc ResourceClient) (*mv1.WatchTopic, error) {
	ri, err := rc.Get(fmt.Sprintf("/management/v1alpha1/watchtopics/%s", name))

	if err == nil {
		wt := &mv1.WatchTopic{}
		err = wt.FromInstance(ri)
		return wt, err
	}

	bts, err := parseWatchTopicTemplate(name, scope)
	if err != nil {
		return nil, err
	}

	return CreateWatchTopic(bts, rc)
}

// parseWatchTopicTemplate parses a WatchTopic from a template
func parseWatchTopicTemplate(name, scope string) ([]byte, error) {
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

	return buf.Bytes(), nil
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

// NewWatchTopic creates a WatchTopic template string
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
