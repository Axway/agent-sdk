package v1

import (
	"testing"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
		{
			"by reference",
			Reference(management.APIServiceGVK(), "my-rd-pods"),
			`metadata.references.name==my-rd-pods;metadata.references.kind==APIService`,
		},
		{
			"by reference or attribute",
			Or(AttrIn("a", "v1", "v2"), Reference(management.APIServiceGVK(), "my-rd-svc")),
			`(attributes.a=in=("v1","v2"),metadata.references.name==my-rd-svc;metadata.references.kind==APIService)`,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			rv := newRSQLVisitor()
			tc.input.Accept(rv)
			if rv.String() != tc.expected {
				t.Errorf("got: %s, expected: %s", rv.String(), tc.expected)
			}
		})
	}
}
