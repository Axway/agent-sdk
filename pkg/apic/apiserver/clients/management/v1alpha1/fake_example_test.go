package management_test

import (
	"testing"

	cv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/api/v1"
	cMgmgt "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	aMgmgt "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

func TestExampleFake(t *testing.T) {
	// can be started with a set of initial resources
	cb, err := cv1.NewFakeClient(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: aMgmgt.EnvironmentGVK(),
			Name:             "environment",
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	// can use a typed client with it
	// here creating a client for Environments
	apisClient, err := cMgmgt.NewAPIServiceClient(cb)
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	// K8SResource is scoped under K8SCluster so I need to use WithScope
	created, err := apisClient.WithScope("environment").Create(&aMgmgt.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			Name: "muhName",
			Attributes: map[string]string{
				"attr": "val",
			},
		},
		Spec: aMgmgt.ApiServiceSpec{
			Description: "",
		},
	})
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	// then I can list it
	list, err := apisClient.WithScope("environment").List(cv1.WithQuery(cv1.AttrIn("attr", "val")))
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	if list[0].Metadata.ID != created.Metadata.ID {
		t.Fatalf("List didn't get what I expected: got %+v, expected %+v ", list[0], created)
	}

	// update the resource and clear attributes
	created.Attributes = map[string]string{}
	// K8SResource is scoped under K8SCluster so I need to use WithScope
	_, err = apisClient.WithScope("environment").Update(created)
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	// the list won't contain the resource anymore
	list, err = apisClient.WithScope("environment").List(cv1.WithQuery(cv1.AttrIn("attr", "val")))
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	if len(list) != 0 {
		t.Fatalf("List didn't shouldn't be empty")
	}
}

func TestExampleFakeUpdate(t *testing.T) {
	// can be started with a set of initial resources
	cb, err := cv1.NewFakeClient(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: aMgmgt.EnvironmentGVK(),
			Name:             "environment",
		},
		Spec: map[string]interface{}{},
	}, &aMgmgt.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{Name: "environment"},
			},
			GroupVersionKind: aMgmgt.EnvironmentGVK(),
			Name:             "myresource",
			Tags:             []string{"existing"},
		},
		Spec: aMgmgt.ApiServiceSpec{},
	})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	// can use a typed client with it
	// here creating a client for K8SResource
	apisResClient, err := cMgmgt.NewAPIServiceClient(cb)
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	update := &aMgmgt.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{Name: "environment"},
			},
			GroupVersionKind: aMgmgt.APIServiceGVK(),
			Name:             "myresource",
			Tags:             []string{"update"},
		},
		Spec: aMgmgt.ApiServiceSpec{},
	}
	// K8SResource is scoped under K8SCluster so I need to use WithScope
	_, err = apisResClient.WithScope("environment").Update(
		update,
		cMgmgt.APIServiceMerge(func(p, n *aMgmgt.APIService) (*aMgmgt.APIService, error) {
			n.Tags = append(n.Tags, p.Tags...)

			return n, nil
		}))
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	// the list won't contain the resource anymore
	list, err := apisResClient.WithScope("environment").List(cv1.WithQuery(cv1.AllTags("existing", "update")))
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	if len(list) == 0 {
		t.Fatalf("List shouldn't be empty")
	}
}
