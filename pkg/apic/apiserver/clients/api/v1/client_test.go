package v1

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	management "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"gopkg.in/h2non/gock.v1"
)

const mockJSONEnv = `{
  "group": "management",
  "apiVersion": "v1alpha1",
  "kind": "Environment",
  "name": "test-env-1",
  "title": "test-env-1",
  "metadata": {
    "id": "e4f1cf5371cd390b0171cd43d7460056",
    "audit": {
      "createTimestamp": "2020-04-30T22:45:07.533+0000",
      "createUserId": "DOSA_531453183cc145adb68ed2d8af625eb2",
      "modifyTimestamp": "2020-04-30T22:45:07.533+0000",
      "modifyUserId": "DOSA_531453183cc145adb68ed2d8af625eb2"
    },
    "resourceVersion": "297",
    "references": []
  },
  "attributes": {
    "attr": "value"
  },
  "tags": ["tag1", "tag2"],
  "spec": {
    "description": "desc"
  }
}`

const mockJSONApiSvc = `{
	"group": "management",
	"apiVersion": "v1alpha1",
	"kind": "APIService",
	"name": "test-api-svc",
	"title": "test-api-svc",
	"metadata": {
		"id": "e4e7efa47287250c017296c54e3f01a6",
		"audit": {
			"createTimestamp": "2020-06-09T01:50:12.548+0000",
			"modifyTimestamp": "2020-06-09T01:50:12.548+0000"
		},
		"scope": {
			"id": "e4e7efa47287250c017296c54dd301a3",
			"kind": "Environment",
			"name": "test-env-1"
		},
		"resourceVersion": "696",
		"references": []
	},
	"attributes": {
		"attr": "value"
	},
	"tags": [
		"atag"
	],
	"spec": {}
}`

var mockEnv = &apiv1.ResourceInstance{}
var mockEnvUpdated = &apiv1.ResourceInstance{}
var mockApiSvc = &apiv1.ResourceInstance{}
var client = &Client{}

func createEnv(client *Client) (*apiv1.ResourceInstance, error) {
	created, err := client.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
			Name:             "test-env-1",
			Title:            "test-env-1",
			Tags:             []string{"atag"},
			Attributes:       map[string]string{"attr": "value"},
		},
		Spec: map[string]interface{}{},
	})
	return created, err
}

func TestMain(m *testing.M) {
	defer gock.Off()
	json.Unmarshal([]byte(mockJSONEnv), mockEnv)
	json.Unmarshal([]byte(mockJSONEnv), mockEnvUpdated)
	mockEnvUpdated.Title = "updated-testenv-title"
	json.Unmarshal([]byte(mockJSONApiSvc), mockApiSvc)
	newClient, err := NewClient(
		"http://localhost:8080/apis",
		BasicAuth(
			"admin",
			"servicesecret",
			"admin",
			"123",
		),
	).ForKind(management.EnvironmentGVK())
	client = newClient
	if err != nil {
		log.Fatalf("Error in test setup: %s", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestUnscoped(t *testing.T) {
	// Create env
	gock.New("http://localhost:8080/apis").
		Post("/management/v1alpha1/environments").
		Reply(201).
		JSON(mockEnv)

	// Update env
	gock.New("http://localhost:8080/apis").
		Put("/management/v1alpha1/environments").
		Reply(200).
		JSON(mockEnvUpdated)

	// Get env by name
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments/test-env-1").
		Reply(200).
		JSON(mockEnv)

	// List envs
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(200).
		JSON([]*apiv1.ResourceInstance{mockEnv})

	// Delete env
	gock.New("http://localhost:8080/apis").
		Delete("/management/v1alpha1/environments/test-env-1").
		Reply(204)

	created, err := createEnv(client)

	if err != nil {
		t.Fatalf("Failed to create: %s", err)
	}

	// Get env by name
	_, err = client.Get(created.Name)
	if err != nil {
		t.Fatalf("Failed to get env by name: %s", err)
	}

	// Update env
	created.Title = "updated-testenv-title"
	updatedEnv, err := client.Update(created)

	if updatedEnv.Title != mockEnvUpdated.Title {
		t.Fatalf("Updated resource name does not match %s. Received %s", mockEnvUpdated.Title, updatedEnv.Title)
	}

	if err != nil {
		t.Fatalf("Failed to update: %s", err)
	}

	// Get all envs
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

func TestScoped(t *testing.T) {
	// Create env
	gock.New("http://localhost:8080/apis").
		Post("/management/v1alpha1/environments").
		Reply(201).
		JSON(mockEnv)

	// Create api service
	gock.New("http://localhost:8080/apis").
		Post("/management/v1alpha1/environments/test-env-1/apiservices").
		Reply(201).
		JSON(mockApiSvc)

	// List api services
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments/test-env-1/apiservices").
		Reply(200).
		JSON([]*apiv1.ResourceInstance{mockApiSvc})

	// Delete api service
	gock.New("http://localhost:8080/apis").
		Delete("/management/v1alpha1/environments/test-env-1/apiservices/test-api-svc").
		Reply(204)

	// Delete env
	gock.New("http://localhost:8080/apis").
		Delete("/management/v1alpha1/environments/test-env-1").
		Reply(204)

	env, err := createEnv(client)

	defer func() {
		err = client.Delete(env)
		if err != nil {
			t.Fatalf("Failed: %s", err)
		}
	}()

	svcClient, err := client.ForKind(management.APIServiceGVK())
	svcClient = svcClient.WithScope(env.Name)

	svc, err := svcClient.Create(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Name:       "test-api-svc",
			Tags:       []string{"atag"},
			Attributes: map[string]string{"attr": "value"},
		},
		Spec: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Failed: %s", err)
	}

	svcs, err := svcClient.List()
	if err != nil {
		t.Fatalf("Failed to list api services: %s", err)
	}
	found := false
	for _, s := range svcs {
		svcClient.Delete(svc)
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

func TestListWithQuery(t *testing.T) {
	// List envs
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		MatchParam("query", "name==test,tags==test").
		Reply(200).
		JSON([]*apiv1.ResourceInstance{mockEnv})

	client.SetQuery("?query=name==test,tags==test")
	_, err := client.List()
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
}

func TestJWTAuth(t *testing.T) {
	token := "abcdefg"
	tenantID := "123456"
	gock.New("http://localhost:8080/apis").
		Post("/management/v1alpha1/environments").
		MatchHeader("Authorization", "Bearer abcdefg").
		MatchHeader("X-Axway-Tenant-Id", tenantID).
		Reply(201).
		JSON(mockEnv)

	client, err := NewClient(
		"http://localhost:8080/apis",
		JWTAuth(
			token,
			tenantID,
		),
	).ForKind(management.EnvironmentGVK())

	if err != nil {
		t.Fatalf("Error creating client with JWT Auth: %s", err)
	}
	_, err = createEnv(client)

	if err != nil {
		t.Fatalf("Error creating env with JWT Auth: %s", err)
	}
}

func TestListError(t *testing.T) {
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(500).
		JSON(mockEnv)

	_, err := client.List()

	if err == nil {
		t.Fatalf("Expected list to fail: %s", err)
	}
}

func TestGetError(t *testing.T) {
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(500).
		JSON(mockEnv)

	_, err := client.Get("name")

	if err == nil {
		t.Fatalf("Expected Update to fail: %s", err)
	}
}

func TestDeleteError(t *testing.T) {
	gock.New("http://localhost:8080/apis").
		Delete("/management/v1alpha1/environments").
		Reply(500).
		JSON(mockEnv)

	err := client.Delete(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
			Name:             "test-env-1",
			Title:            "test-env-1",
			Tags:             []string{"atag"},
			Attributes:       map[string]string{"attr": "value"},
		},
		Spec: map[string]interface{}{},
	})

	if err == nil {
		t.Fatalf("Expected delete to fail: %s", err)
	}
}

func TestCreateError(t *testing.T) {
	gock.New("http://localhost:8080/apis").
		Post("/management/v1alpha1/environments").
		Reply(500).
		JSON(mockEnv)

	_, err := createEnv(client)

	if err == nil {
		t.Fatalf("Expected create to fail: %s", err)
	}
}

func TestUpdateError(t *testing.T) {
	gock.New("http://localhost:8080/apis").
		Put("/management/v1alpha1/environments").
		Reply(500).
		JSON(mockEnv)

	_, err := client.Update(&apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
			Name:             "test-env-1",
			Title:            "test-env-1",
			Tags:             []string{"atag"},
			Attributes:       map[string]string{"attr": "value"},
		},
		Spec: map[string]interface{}{},
	})

	if err == nil {
		t.Fatalf("Expected update to fail: %s", err)
	}
}
