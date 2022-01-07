//go:build withapiserver
// +build withapiserver

// Tests in this file are not run by default
// They need an apiserver started on localhost:8080
// to run apply the "withapiserver" tag to the test command:
// go test --tags withapiserver

package v1_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	. "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

func TestQueries(t *testing.T) {
	cb := NewClient(
		"http://localhost:8080/apis",
		UserAgent("fake-agent"),
		BasicAuth(
			"admin",
			"servicesecret",
			"admin", "123",
		),
	)

	cEnv, err := cb.ForKind(management.EnvironmentGVK())
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	env1, err := cEnv.Create(context.Background(), &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: fmt.Sprintf("env1.%d", time.Now().Unix()),
			Attributes: map[string]string{
				"attr1":    "val1",
				"attr":     "val",
				"diffattr": "val1",
			},
			Tags: []string{
				"tag", "tag1",
			},
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}
	defer func() {
		cEnv.Delete(context.Background(), env1)
	}()

	env2, err := cEnv.Create(context.Background(), &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: fmt.Sprintf("env2.%d", time.Now().Unix()),
			Attributes: map[string]string{
				"attr2":    "val2",
				"attr":     "val",
				"diffattr": "val2",
			},
			Tags: []string{
				"tag", "tag2",
			},
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}
	defer func() {
		cEnv.Delete(context.Background(), env2)
	}()

	testCases := []struct {
		name     string
		query    QueryNode
		expected []string
	}{{
		"common attribute and value",
		AttrIn("attr", "val"),
		[]string{env1.Name, env2.Name},
	}, {
		"common tag",
		TagsIn("tag"),
		[]string{env1.Name, env2.Name},
	}, {
		"tag with one match",
		TagsIn("tag1"),
		[]string{env1.Name},
	}, {
		"two tags",
		TagsIn("tag1", "tag2"),
		[]string{env1.Name, env2.Name},
	}, {
		"attribute with two values",
		AttrIn("diffattr", "val1"),
		[]string{env1.Name},
	}, {
		"any attr",
		AnyAttr(map[string]string{"attr1": "val1", "attr2": "val2"}),
		[]string{env1.Name, env2.Name},
	}, {
		"all attr",
		AllAttr(map[string]string{"attr1": "val1", "diffattr": "val1"}),
		[]string{env1.Name},
	}, {
		"all attr and one tag",
		And(AllAttr(map[string]string{"attr1": "val1", "diffattr": "val1"}), TagsIn("tag")),
		[]string{env1.Name},
	}, {
		"all attr and one tag no result",
		And(AllAttr(map[string]string{"attr1": "val1", "diffattr": "val1"}), TagsIn("tag2")),
		[]string{},
	}, {
		"all attr or one tag",
		Or(AllAttr(map[string]string{"attr1": "val1", "diffattr": "val1"}), TagsIn("tag2")),
		[]string{env1.Name, env2.Name},
	},
	}

	for i, _ := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ris, err := cEnv.List(context.Background(), WithQuery(tc.query))
			if err != nil {
				t.Errorf("Failed due: %s", err)
			}

			names := make([]string, len(ris))

			for i, ri := range ris {
				names[i] = ri.Name
			}

			if !reflect.DeepEqual(tc.expected, names) {
				t.Errorf("Gots %+v, expected %+v", names, tc.expected)
			}
		})
	}
}
