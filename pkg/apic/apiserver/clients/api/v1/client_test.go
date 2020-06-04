package v1

import (
	"testing"

	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	management "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// needs apiserver started to run
func TestUnscoped(t *testing.T) {
	c, err := NewClient(
		"http://localhost:8080/apis",
		BasicAuth(
			"admin",
			"servicesecret",
			"admin",
			"123",
		),
	).ForKind(management.EnvironmentGVK())

	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	created, err := c.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
			Name:             "testenv1",
			Tags:             []string{"atag"},
			Attributes:       map[string]string{"attr": "value"},
		},
		Spec: map[string]interface{}{},
	})

	envList, err := c.List()
	if err != nil {
		t.Fatalf("Failed due: %s", err)
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

	err = c.Delete(created)
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}
}

// needs apiserver started to run
func TestScoped(t *testing.T) {
	cb := NewClient(
		"http://localhost:8080/apis",
		BasicAuth(
			"admin",
			"servicesecret",
			"admin",
			"123",
		),
	)

	envClient, err := cb.ForKind(management.EnvironmentGVK())
	if err != nil {
		t.Fatalf("Failed due: %s", err)
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
			t.Fatalf("Failed due: %s", err)
		}
	}()

	svcClient, err := cb.ForKind(management.APIServiceGVK())
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
		t.Fatalf("Failed due: %s", err)
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
		t.Fatalf("Failed due: %s", err)
	}

}
