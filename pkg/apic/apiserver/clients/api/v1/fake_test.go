package v1_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/api/v1"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

func TestFakeUnscoped(t *testing.T) {
	cb, err := v1.NewFakeClient(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.K8SClusterGVK(),
			Name:             "muhName",
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	k8sClient, err := cb.ForKind(management.K8SClusterGVK())
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}
	k8s, err := k8sClient.Get("muhName")
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}
	t.Log(k8s)
}

func TestAddFakeUnscoped(t *testing.T) {
	cb, err := v1.NewFakeClient(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.K8SClusterGVK(),
			Name:             "muhName",
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	k8sClient, err := cb.ForKind(management.K8SClusterGVK())
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	_, err = k8sClient.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.K8SClusterGVK(),
			Name:             "muhName",
		},
		Spec: map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("Failed due to: expected error")
	}

	_, err = k8sClient.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.K8SClusterGVK(),
			Name:             "muhSecondName",
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}
}

func TestFakeScoped(t *testing.T) {
	cb, err := v1.NewFakeClient(
		&apiv1.ResourceInstance{
			ResourceMeta: apiv1.ResourceMeta{
				GroupVersionKind: management.K8SClusterGVK(),
				Name:             "muhName",
			},
			Spec: map[string]interface{}{},
		},
		&apiv1.ResourceInstance{
			ResourceMeta: apiv1.ResourceMeta{
				GroupVersionKind: management.K8SResourceGVK(),
				Name:             "muhResource",
				Metadata: apiv1.Metadata{
					Scope: apiv1.MetadataScope{
						Name: "muhName",
					},
				},
			},
			Spec: map[string]interface{}{},
		},
	)
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	noScope, err := cb.ForKind(management.K8SResourceGVK())
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}
	_, err = noScope.WithScope("muhName").Get("muhResource")
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	ri, err := noScope.WithScope("muhName").Update(
		&apiv1.ResourceInstance{
			ResourceMeta: apiv1.ResourceMeta{
				GroupVersionKind: management.K8SResourceGVK(),
				Name:             "muhResource",
				Metadata: apiv1.Metadata{
					Scope: apiv1.MetadataScope{
						Name: "muhName",
					},
				},
				Attributes: map[string]string{"attribute": "value"},
				Tags:       []string{"tag"},
			},
			Spec: map[string]interface{}{},
		})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	t.Logf("%+v", ri)
}

func TestFakeQueries(t *testing.T) {
	cb, err := v1.NewFakeClient(
		&apiv1.ResourceInstance{
			ResourceMeta: apiv1.ResourceMeta{
				Name:             "env1",
				GroupVersionKind: management.EnvironmentGVK(),
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
		},
		&apiv1.ResourceInstance{
			ResourceMeta: apiv1.ResourceMeta{
				Name:             "env2",
				GroupVersionKind: management.EnvironmentGVK(),
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
		},
	)
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	cEnv, err := cb.ForKind(management.EnvironmentGVK())
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	testCases := []struct {
		name     string
		query    v1.QueryNode
		expected []string
	}{{
		"query names",
		v1.Names("env1"),
		[]string{"env1"},
	},

		{
			"common attribute and value",
			v1.AttrIn("attr", "val"),
			[]string{"env1", "env2"},
		}, {
			"common tag",
			v1.TagsIn("tag"),
			[]string{"env1", "env2"},
		}, {
			"tag with one match",
			v1.TagsIn("tag1"),
			[]string{"env1"},
		}, {
			"two tags",
			v1.TagsIn("tag1", "tag2"),
			[]string{"env1", "env2"},
		}, {
			"attribute with two values",
			v1.AttrIn("diffattr", "val1"),
			[]string{"env1"},
		}, {
			"any attr",
			v1.AnyAttr(map[string]string{"attr1": "val1", "attr2": "val2"}),
			[]string{"env1", "env2"},
		}, {
			"all attr",
			v1.AllAttr(map[string]string{"attr1": "val1", "diffattr": "val1"}),
			[]string{"env1"},
		}, {
			"all attr and one tag",
			v1.And(v1.AllAttr(map[string]string{"attr1": "val1", "diffattr": "val1"}), v1.TagsIn("tag")),
			[]string{"env1"},
		}, {
			"all attr and one tag no result",
			v1.And(v1.AllAttr(map[string]string{"attr1": "val1", "diffattr": "val1"}), v1.TagsIn("tag2")),
			[]string{},
		}, {
			"all attr or one tag",
			v1.Or(v1.AllAttr(map[string]string{"attr1": "val1", "diffattr": "val1"}), v1.TagsIn("tag2")),
			[]string{"env1", "env2"},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ris, err := cEnv.List(v1.WithQuery(tc.query))
			if err != nil {
				t.Errorf("Failed due: %s", err)
			}

			names := make([]string, len(ris))

			for i, ri := range ris {
				names[i] = ri.Name
			}

			sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })

			if !reflect.DeepEqual(tc.expected, names) {
				t.Errorf("Got %+v, expected %+v", names, tc.expected)
			}
		})
	}
}

func withAttr(attr map[string]string) func(*apiv1.ResourceInstance) {
	return func(ri *apiv1.ResourceInstance) {
		ri.Attributes = attr
	}
}

func withTags(tags []string) func(*apiv1.ResourceInstance) {
	return func(ri *apiv1.ResourceInstance) {
		ri.Tags = tags
	}
}

type buildOption func(*apiv1.ResourceInstance)

func env(name string, opts ...buildOption) *apiv1.ResourceInstance {
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             name,
			GroupVersionKind: management.EnvironmentGVK(),
		},
	}

	for _, o := range opts {
		o(ri)
	}

	return ri
}

func apisvc(name string, scopeName string, opts ...buildOption) *apiv1.ResourceInstance {
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             name,
			GroupVersionKind: management.APIServiceGVK(),
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Name: scopeName,
				},
			},
		},
	}

	for _, o := range opts {
		o(ri)
	}

	return ri
}

func TestFakeUpdateMerge(t *testing.T) {
	old := &management.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{},
			Name:             "name",
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Name: "myenv",
				},
				References: []apiv1.Reference{},
			},
			Tags: []string{"old"},
		},
	}

	update := &management.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{},
			Name:             "name",
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Name: "myenv",
				},
				References: []apiv1.Reference{},
			},
			Attributes: map[string]string{},
			Tags:       []string{"new"},
		},
	}

	merged := &management.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{},
			Name:             "name",
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Name: "myenv",
				},
				References: []apiv1.Reference{},
			},
			Attributes: map[string]string{},
			Tags:       []string{"old", "new"},
		},
	}

	testCases := []struct {
		name     string
		old      apiv1.Interface
		update   apiv1.Interface
		mf       v1.MergeFunc
		expected apiv1.Interface
	}{{
		name:   "it's a create",
		old:    nil,
		update: update,
		mf: func(fetched, new apiv1.Interface) (apiv1.Interface, error) {
			return new, nil
		},
		expected: update,
	}, {
		name:   "it's an overwrite",
		old:    old,
		update: update,
		mf: func(fetched, new apiv1.Interface) (apiv1.Interface, error) {
			return new, nil
		},
		expected: update,
	}, {
		name:   "it's a merge",
		old:    old,
		update: update,
		mf: func(fetched, new apiv1.Interface) (apiv1.Interface, error) {
			ri, err := fetched.AsInstance()
			if err != nil {
				return nil, err
			}
			ri.Tags = append(ri.Tags, new.GetTags()...)

			return ri, nil
		},
		expected: merged,
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			b, err := v1.NewFakeClient(&management.Environment{ResourceMeta: apiv1.ResourceMeta{Name: "myenv"}})
			if err != nil {
				t.Fatal(err)
			}

			us, err := b.ForKind(management.APIServiceGVK())
			if err != nil {
				t.Fatal(err)
			}

			if tc.old != nil {
				i, err := tc.old.AsInstance()
				if err != nil {
					t.Fatalf("Failed due: %s", err)
				}
				_, err = us.Create(i)
				if err != nil {
					t.Error("Failed to create old resource: ", err)
				}
			}
			i, err := tc.update.AsInstance()
			if err != nil {
				t.Fatalf("Failed due: %s", err)
			}

			r, err := us.Update(i, v1.Merge(tc.mf))
			if err != nil {
				t.Errorf("Failed to update: %s", err)
				return
			}

			if !reflect.DeepEqual(r.GetTags(), tc.expected.GetTags()) {
				t.Errorf("Expected tags %+v; Got: %+v ", tc.expected.GetTags(), r.GetTags())
			}

		})
	}
}

func TestFake(t *testing.T) {
	testCases := []struct {
		name           string
		init           []apiv1.Interface
		add            []*apiv1.ResourceInstance
		update         []*apiv1.ResourceInstance
		delete         []*apiv1.ResourceInstance
		queryClient    func(v1.Base) v1.Scoped
		query          v1.QueryNode
		expectedBefore []string
		expectedAfter  []string
	}{{
		"attr list after delete",
		[]apiv1.Interface{
			env("env1"),
			apisvc("svc1", "env1", withAttr(map[string]string{"attr": "val"})),
		},
		[]*apiv1.ResourceInstance{},
		[]*apiv1.ResourceInstance{},
		[]*apiv1.ResourceInstance{apisvc("svc1", "env1")},
		func(b v1.Base) v1.Scoped { c, _ := b.ForKind(management.APIServiceGVK()); return c.WithScope("env1") },
		v1.AttrIn("attr", "val"),
		[]string{"svc1"},
		[]string{},
	}, {
		"attr list after update",
		[]apiv1.Interface{
			env("env1"),
			apisvc("svc1", "env1"),
		},
		[]*apiv1.ResourceInstance{},
		[]*apiv1.ResourceInstance{apisvc("svc1", "env1", withAttr(map[string]string{"attr": "val"}))},
		[]*apiv1.ResourceInstance{},
		func(b v1.Base) v1.Scoped { c, _ := b.ForKind(management.APIServiceGVK()); return c.WithScope("env1") },
		v1.AttrIn("attr", "val"),
		[]string{},
		[]string{"svc1"},
	}, {
		"tags list after delete",
		[]apiv1.Interface{
			env("env1"),
			apisvc("svc1", "env1", withTags([]string{"tag1"})),
		},
		[]*apiv1.ResourceInstance{},
		[]*apiv1.ResourceInstance{},
		[]*apiv1.ResourceInstance{apisvc("svc1", "env1")},
		func(b v1.Base) v1.Scoped { c, _ := b.ForKind(management.APIServiceGVK()); return c.WithScope("env1") },
		v1.TagsIn("tag1"),
		[]string{"svc1"},
		[]string{},
	}, {
		"tags list after update",
		[]apiv1.Interface{
			env("env1"),
			apisvc("svc1", "env1"),
		},
		[]*apiv1.ResourceInstance{},
		[]*apiv1.ResourceInstance{apisvc("svc1", "env1", withTags([]string{"tag1"}))},
		[]*apiv1.ResourceInstance{},
		func(b v1.Base) v1.Scoped { c, _ := b.ForKind(management.APIServiceGVK()); return c.WithScope("env1") },
		v1.TagsIn("tag1"),
		[]string{},
		[]string{"svc1"},
	}, {
		"attribute and tags list after add",
		[]apiv1.Interface{
			env("env1"),
			apisvc("svc1", "env1", withAttr(map[string]string{"attr1": "val1"}), withTags([]string{"tag1"})),
		},
		[]*apiv1.ResourceInstance{apisvc("svc2", "env1", withAttr(map[string]string{"attr1": "val1"}), withTags([]string{"tag1"}))},
		[]*apiv1.ResourceInstance{},
		[]*apiv1.ResourceInstance{},
		func(b v1.Base) v1.Scoped { c, _ := b.ForKind(management.APIServiceGVK()); return c.WithScope("env1") },
		v1.TagsIn("tag1"),
		[]string{"svc1"},
		[]string{"svc1", "svc2"},
	},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			fk, err := v1.NewFakeClient(tc.init...)
			if err != nil {
				t.Fatalf("Failed due: %s", err)
			}

			ris, err := tc.queryClient(fk).List(v1.WithQuery(tc.query))
			if err != nil {
				t.Fatalf("List query failed due: %s", err)
			}

			names := make([]string, len(ris))
			for i, ri := range ris {
				names[i] = ri.Name
			}

			sort.Strings(tc.expectedBefore)
			sort.Strings(names)

			if !reflect.DeepEqual(tc.expectedBefore, names) {
				t.Fatalf("Before: got %+v, expected %+v", names, tc.expectedBefore)
			}

			for _, ri := range tc.add {
				c, err := fk.ForKind(ri.GroupVersionKind)
				if err != nil {
					t.Fatalf("Failed to add %+v: %s", ri, err)
				}

				s := c.(v1.Scoped)

				if ri.Metadata.Scope.Name != "" {
					s = c.WithScope(ri.Metadata.Scope.Name)
				}

				_, err = s.Create(ri)
				if err != nil {
					t.Fatalf("Failed to add %+v: %s", ri, err)
				}
			}

			for _, ri := range tc.update {
				c, err := fk.ForKind(ri.GroupVersionKind)
				if err != nil {
					t.Fatalf("Failed to update %+v: %s", ri, err)
				}

				s := c.(v1.Scoped)

				if ri.Metadata.Scope.Name != "" {
					s = c.WithScope(ri.Metadata.Scope.Name)
				}

				_, err = s.Update(ri)
				if err != nil {
					t.Fatalf("Failed to update %+v: %s", ri, err)
				}
			}

			for _, ri := range tc.delete {
				c, err := fk.ForKind(ri.GroupVersionKind)
				if err != nil {
					t.Fatalf("Failed to delete %+v: %s", ri, err)
				}

				s := c.(v1.Scoped)

				if ri.Metadata.Scope.Name != "" {
					s = c.WithScope(ri.Metadata.Scope.Name)
				}

				err = s.Delete(ri)
				if err != nil {
					t.Fatalf("Failed to delete %+v: %s", ri, err)
				}
			}

			ris, err = tc.queryClient(fk).List(v1.WithQuery(tc.query))
			if err != nil {
				t.Errorf("Failed due: %s", err)
			}

			names = make([]string, len(ris))
			for i, ri := range ris {
				names[i] = ri.Name
			}

			sort.Strings(tc.expectedAfter)
			sort.Strings(names)

			if !reflect.DeepEqual(tc.expectedAfter, names) {
				t.Errorf("After: got %+v, expected %+v", names, tc.expectedAfter)
			}

		})
	}

}

func TestFakeEvents(t *testing.T) {
	fk, err := v1.NewFakeClient()
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	ri := env("env1")

	fk.SetHandler(v1.EventHandlerFunc(func(e *apiv1.Event) {
		assert.NotNil(t, e)
	}))

	c, err := fk.ForKind(management.EnvironmentGVK())
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	_, err = c.Create(ri)
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	_, err = c.Update(ri)
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	err = c.Delete(ri)
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}
}

func TestFakeScopedDeleteEvents(t *testing.T) {
	fk, err := v1.NewFakeClient(env("env1"), apisvc("svc1", "env1"))
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	c, err := fk.ForKind(management.EnvironmentGVK())
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	fk.SetHandler(v1.EventHandlerFunc(func(e *apiv1.Event) {
		assert.NotNil(t, e)
		if e.Type != apiv1.ResourceEntryDeletedEvent {
			t.Errorf("Expected %s event", apiv1.ResourceEntryDeletedEvent)
		}

		if e.Payload.Name != "svc1" {
			t.Errorf("Unexpected resource name. Expected svc1 got: %+v", e)
		}

		fk.SetHandler(v1.EventHandlerFunc(func(e *apiv1.Event) {
			if e.Type != apiv1.ResourceEntryDeletedEvent {
				t.Errorf("Expected %s event", apiv1.ResourceEntryDeletedEvent)
			}

			if e.Payload.Name != "env1" {
				t.Errorf("Unexpected resource name. Expected env1 got %+v", e)
			}
		}))
	}))
	err = c.Delete(env("env1"))

	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}
}
