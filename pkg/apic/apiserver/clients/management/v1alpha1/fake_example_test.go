package v1alpha1_test

import (
	"testing"

	cv1 "github.com/Axway/agents-sdk/pkg/apic/apiserver/clients/api/v1"
	cMgmgt "github.com/Axway/agents-sdk/pkg/apic/apiserver/clients/management/v1alpha1"
	apiv1 "github.com/Axway/agents-sdk/pkg/apic/apiserver/models/api/v1"
	aMgmgt "github.com/Axway/agents-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

func TestExampleFake(t *testing.T) {
	// can be started with a set of initial resources
	cb, err := cv1.NewFakeClient(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: aMgmgt.K8SClusterGVK(),
			Name:             "muhCluster",
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	// can use a typed client with it
	// here creating a client for K8SResource
	k8sResClient, err := cMgmgt.NewK8SResourceClient(cb)
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	// K8SResource is scoped under K8SCluster so I need to use WithScope
	created, err := k8sResClient.WithScope("muhCluster").Create(&aMgmgt.K8SResource{
		ResourceMeta: apiv1.ResourceMeta{
			Name: "muhName",
			Attributes: map[string]string{
				"attr": "val",
			},
		},
		Spec: aMgmgt.K8SResourceSpec{
			ResourceSpec: map[string]interface{}{},
		},
	})
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	// then I can list it
	list, err := k8sResClient.WithScope("muhCluster").List(cv1.WithQuery(cv1.AttrIn("attr", "val")))
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	if list[0].Metadata.ID != created.Metadata.ID {
		t.Fatalf("List didn't get what I expected: got %+v, expected %+v ", list[0], created)
	}

	// update the resource and clear attributes
	created.Attributes = map[string]string{}
	// K8SResource is scoped under K8SCluster so I need to use WithScope
	_, err = k8sResClient.WithScope("muhCluster").Update(created)
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	// the list won't contain the resource anymore
	list, err = k8sResClient.WithScope("muhCluster").List(cv1.WithQuery(cv1.AttrIn("attr", "val")))
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
			GroupVersionKind: aMgmgt.K8SClusterGVK(),
			Name:             "muhCluster",
		},
		Spec: map[string]interface{}{},
	}, &aMgmgt.K8SResource{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{Name: "muhCluster"},
			},
			GroupVersionKind: aMgmgt.K8SClusterGVK(),
			Name:             "myresource",
			Tags:             []string{"existing"},
		},
		Spec: aMgmgt.K8SResourceSpec{},
	})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	// can use a typed client with it
	// here creating a client for K8SResour ce
	k8sResClient, err := cMgmgt.NewK8SResourceClient(cb)
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	update := &aMgmgt.K8SResource{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{Name: "muhCluster"},
			},
			GroupVersionKind: aMgmgt.K8SClusterGVK(),
			Name:             "myresource",
			Tags:             []string{"update"},
		},
		Spec: aMgmgt.K8SResourceSpec{},
	}
	// K8SResource is scoped under K8SCluster so I need to use WithScope
	_, err = k8sResClient.WithScope("muhCluster").Update(
		update,
		cMgmgt.APIServiceMerge(func(p, n *aMgmgt.APIService) (*aMgmgt.APIService, error) {
			n.Tags = append(n.Tags, p.Tags...)

			return n, nil
		}))
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	// the list won't contain the resource anymore
	list, err := k8sResClient.WithScope("muhCluster").List(cv1.WithQuery(cv1.AllTags("existing", "update")))
	if err != nil {
		t.Fatalf("Failed due to: %s", err)
	}

	if len(list) == 0 {
		t.Fatalf("List shouldn't be empty")
	}
}
