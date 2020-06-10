package v1_test

import (
	"testing"

	v1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/clients/api/v1"
	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

func TestFakeUnscoped(t *testing.T) {
	cb, err := v1.NewFakeClient(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: v1alpha1.K8SClusterGVK(),
			Name:             "muhName",
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	k8sClient, err := cb.ForKind(v1alpha1.K8SClusterGVK())
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
			GroupVersionKind: v1alpha1.K8SClusterGVK(),
			Name:             "muhName",
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	k8sClient, err := cb.ForKind(v1alpha1.K8SClusterGVK())
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	_, err = k8sClient.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: v1alpha1.K8SClusterGVK(),
			Name:             "muhName",
		},
		Spec: map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("Failed due to: expected error")
	}

	_, err = k8sClient.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: v1alpha1.K8SClusterGVK(),
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
				GroupVersionKind: v1alpha1.K8SClusterGVK(),
				Name:             "muhName",
			},
			Spec: map[string]interface{}{},
		},
		&apiv1.ResourceInstance{
			ResourceMeta: apiv1.ResourceMeta{
				GroupVersionKind: v1alpha1.K8SResourceGVK(),
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

	noScope, err := cb.ForKind(v1alpha1.K8SResourceGVK())
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}
	ri, err := noScope.WithScope("muhName").Get("muhResource")
	if err != nil {
		t.Fatal("Failed due to: ", err)
	}

	ri, err = noScope.WithScope("muhName").Update(
		&apiv1.ResourceInstance{
			ResourceMeta: apiv1.ResourceMeta{
				GroupVersionKind: v1alpha1.K8SResourceGVK(),
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
