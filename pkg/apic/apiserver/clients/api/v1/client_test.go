package v1

import (
	"fmt"
	"testing"
	"time"

	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	management "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// needs apiserver started to run
func TestUnscoped(t *testing.T) {
	client, err := NewClient(
		"http://localhost:8080/apis",
		BasicAuth(
			"admin",
			"servicesecret",
			"admin",
			"123",
		),
	).ForKind(management.EnvironmentGVK())

	// HTTPClient(client.ClientBase.client)()

	if err != nil {
		t.Fatalf("Failed: %s", err)
	}
	created, err := client.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
			Name:             fmt.Sprintf("test-env-%d", time.Now().Unix()),
			Title:            fmt.Sprintf("test-env-%d", time.Now().Unix()),
			Tags:             []string{"atag"},
			Attributes:       map[string]string{"attr": "value"},
		},
		Spec: map[string]interface{}{},
	})

	if err != nil {
		t.Fatalf("Failed to create: %s", err)
	}

	_, err = client.Get(created.Name)
	if err != nil {
		t.Fatalf("Failed to get env by name: %s", err)
	}

	created.Title = "updated-testenv-title"
	_, err = client.Update(created)

	if err != nil {
		t.Fatalf("Failed to update: %s", err)
	}

	envList, err := client.List()
	if err != nil {
		t.Fatalf("Failed to list environments: %s", err)
	}

	found := false
	for _, env := range envList {
		if env.Name == created.Name {
			found = true
			t.Log("Found in list: ", env)
			break
		}
	}

	if !found {
		t.Fatalf("Cannot find created environment %v", created)
	}

	err = client.Delete(created)
	if err != nil {
		t.Fatalf("Failed to delete: %s", err)
	}
}

// needs apiserver started to run
func TestScoped(t *testing.T) {
	client := NewClient(
		"http://localhost:8080/apis",
		BasicAuth(
			"admin",
			"servicesecret",
			"admin",
			"123",
		),
	)

	envClient, err := client.ForKind(management.EnvironmentGVK())
	if err != nil {
		t.Fatalf("Failed: %s", err)
	}

	env, err := envClient.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
			Name:             "testenv1",
			Tags:             []string{"atag"},
			Attributes:       map[string]string{"attr": "value"},
		},
		Spec: map[string]interface{}{},
	})
	defer func() {
		err = envClient.Delete(env)
		if err != nil {
			t.Fatalf("Failed: %s", err)
		}
	}()

	svcClient, err := client.ForKind(management.APIServiceGVK())
	svcClient = svcClient.WithScope(env.Name)

	svc, err := svcClient.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Name:       "testapsvc",
			Tags:       []string{"atag"},
			Attributes: map[string]string{"attr": "value"},
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Failed: %s", err)
	}

	svcs, err := svcClient.List()
	found := false
	for _, s := range svcs {
		if s.Name == svc.Name {
			t.Logf("Found created svc %v", s)

			found = true
			return
		}
	}
	if !found {
		t.Fatalf("Cannot find created service %v", svc)
	}

	err = svcClient.Delete(svc)
	if err != nil {
		t.Fatalf("Failed: %s", err)
	}

}
