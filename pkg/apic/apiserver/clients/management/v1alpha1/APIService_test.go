package v1alpha1

import (
	"fmt"
	"testing"
	"time"

	v1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/client/api/v1"
	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	management "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

func TestAPIService(t *testing.T) {
	cb := v1.NewClient(
		"http://localhost:8080/apis",
		v1.BasicAuth(
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
			Name:             fmt.Sprintf("env%d", time.Now().Unix()),
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

	svcClient, err := NewAPIServiceClient(cb)
	svcClient = svcClient.WithScope(env.Name)

	svc, err := svcClient.Create(&v1alpha1.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			Name:       "testapsvc",
			Tags:       []string{"atag"},
			Attributes: map[string]string{"attr": "value"},
		},
		Spec: v1alpha1.ApiServiceSpec{
			Description: "hey",
		},
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
