package v1_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	. "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/clients/api/v1"
	v1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

func TestRSQL(t *testing.T) {
	testCases := []struct {
		name     string
		input    QueryNode
		expected string
	}{
		{
			"one attribute, one value",
			AttrIn("a", "v"),
			`attributes.a=="v"`,
		},
		{
			"one attribute, two values",
			AttrIn("a", "v1", "v2"),
			`attributes.a=in=("v1","v2")`,
		},
		{
			"one tag",
			TagsIn("t"),
			`tags=="t"`,
		},
		{
			"three tags",
			TagsIn("t1", "t2", "t3"),
			`tags=in=("t1","t2","t3")`,
		},
		{
			"one attribute, two values ored with three tags",
			Or(AttrIn("a", "v1", "v2"), TagsIn("t1", "t2", "t3")),
			`(attributes.a=in=("v1","v2"),tags=in=("t1","t2","t3"))`,
		},
		{
			"one attribute, one values anded with three tags",
			And(AttrIn("a", "v1"), TagsIn("t1", "t2", "t3")),
			`(attributes.a=="v1";tags=in=("t1","t2","t3"))`,
		},
		{
			"all two attributes",
			AllAttr(map[string]string{"a1": "v1", "a2": "v2"}),
			`(attributes.a1=="v1";attributes.a2=="v2")`,
		},
		{
			"any three attributes",
			AnyAttr(map[string]string{"a1": "v1", "a2": "v2", "a3": "v3"}),
			`(attributes.a1=="v1",attributes.a2=="v2",attributes.a3=="v3")`,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			rv := &RSQLVisitor{}
			tc.input.Accept(rv)
			if rv.String() != tc.expected {
				t.Errorf("got: %s, expected: %s", rv.String(), tc.expected)
			}
		})
	}
}

func TestQuery(t *testing.T) {
	cb := NewClient(
		"http://localhost:8080/apis",
		BasicAuth(
			"admin",
			"servicesecret",
			"admin", "123",
		),
	)

	cEnv, err := cb.ForKind(v1alpha1.EnvironmentGVK())
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	env1, err := cEnv.Create(&v1.ResourceInstance{
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
		cEnv.Delete(env1)
	}()

	env2, err := cEnv.Create(&v1.ResourceInstance{
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
		cEnv.Delete(env2)
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
			ris, err := cEnv.List(WithQuery(tc.query))
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
