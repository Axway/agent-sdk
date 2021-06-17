package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

const privKey = "testdata/privatekey"
const pubKey = "testdata/publickey"
const uaHeader = "fake-agent"

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
var mockAPISvc = &apiv1.ResourceInstance{}
var client = &Client{}

func createEnv(client Unscoped) (*apiv1.ResourceInstance, error) {
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
	json.Unmarshal([]byte(mockJSONEnv), mockEnv)
	json.Unmarshal([]byte(mockJSONEnv), mockEnvUpdated)
	mockEnvUpdated.Title = "updated-testenv-title"
	json.Unmarshal([]byte(mockJSONApiSvc), mockAPISvc)

	newClient, err := NewClient(
		"http://localhost:8080/apis",
		UserAgent(uaHeader),
		BasicAuth(
			"admin",
			"servicesecret",
			"admin",
			"123",
		),
	).ForKind(management.EnvironmentGVK())
	client = newClient.(*Client)
	if err != nil {
		log.Fatalf("Error in test setup: %s", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestUnscoped(t *testing.T) {
	defer gock.Off()
	// Create env
	gock.New("http://localhost:8080/apis").
		Post("/management/v1alpha1/environments").
		MatchHeader("User-Agent", "fake-agent").
		Reply(201).
		JSON(mockEnv)

	gock.New("http://localhost:8080/apis").
		Put("/management/v1alpha1/environments/test-env-1").
		Reply(200).
		JSON(mockEnvUpdated)

	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments/test-env-1").
		Reply(200).
		JSON(mockEnv)

	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(200).
		JSON([]*apiv1.ResourceInstance{mockEnv})

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
	defer gock.Off()
	// Create env
	gock.New("http://localhost:8080/apis").
		Post("/management/v1alpha1/environments").
		Reply(201).
		JSON(mockEnv)

	gock.New("http://localhost:8080/apis").
		Post("/management/v1alpha1/environments/test-env-1/apiservices").
		Reply(201).
		JSON(mockAPISvc)

	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments/test-env-1/apiservices").
		Reply(200).
		JSON([]*apiv1.ResourceInstance{mockAPISvc})

	gock.New("http://localhost:8080/apis").
		Delete("/management/v1alpha1/environments/test-env-1/apiservices/test-api-svc").
		Reply(204)

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
	if err != nil {
		t.Fatalf("Failed: %s", err)
	}
	svcClient = svcClient.WithScope(env.Name).(*Client)

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
	defer gock.Off()
	// List envs
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		MatchHeader("User-Agent", uaHeader).
		MatchParam("query", `(tags=="test";attributes.attr==("val"))`).Reply(200).
		JSON([]*apiv1.ResourceInstance{mockEnv, mockEnv})

	_, err := client.List(WithQuery(And(TagsIn("test"), AttrIn("attr", "val"))))
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
}

func Test_listAll(t *testing.T) {
	// Follow the link headers
	defer gock.Off()
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(200).
		AddHeader("Link", "</apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=2>; rel=\"next\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=1>; rel=\"self\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=1>; rel=\"first\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=4>; rel=\"last\"").
		JSON([]*apiv1.ResourceInstance{mockEnv})

	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(200).
		AddHeader("Link", "</apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=1>; rel=\"self\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=1>; rel=\"first\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=4>; rel=\"last\"").
		JSON([]*apiv1.ResourceInstance{mockEnv})

	items, err := client.List(WithQuery(TagsIn("test")))
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	assert.Equal(t, 2, len(items))

	// handle an error from the client
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(500).
		SetError(&url.Error{})

	_, err = client.List()
	assert.NotNil(t, err)

	// handle a successful response, but a 500 error
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(500)

	_, err = client.List()
	assert.NotNil(t, err)

	// handle a successful request, then a failed request
	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(200).
		AddHeader("Link", "</apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=2>; rel=\"next\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=1>; rel=\"self\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=1>; rel=\"first\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=4>; rel=\"last\"").
		JSON([]*apiv1.ResourceInstance{mockEnv})

	gock.New("http://localhost:8080/apis").
		Get("/management/v1alpha1/environments").
		Reply(500).
		AddHeader("Link", "</apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=1>; rel=\"self\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=1>; rel=\"first\", </apis/management/v1alpha1/environments?pageSize=20&query=tags%3D%3D%22test%22&page=4>; rel=\"last\"").
		JSON([]*apiv1.ResourceInstance{})

	items, err = client.List()
	assert.Equal(t, 0, len(items))
	assert.NotNil(t, err)
}

func TestJWTAuth(t *testing.T) {
	tenantId := "1234"
	c := NewClient("http://localhost:8080",
		JWTAuth(
			tenantId,
			privKey,
			pubKey,
			"",
			"http://localhost:8080/auth/realms/Broker/protocol/openid-connect/token",
			"http://localhost:8080/auth/realms/Broker",
			"DOSA_1234",
			10*time.Second,
		),
		UserAgent(uaHeader),
	)

	assert.Equal(t, uaHeader, c.userAgent)

	jAuthStruct, ok := c.auth.(*jwtAuth)

	assert.True(t, ok)
	jAuthStruct.tokenGetter = apic.MockTokenGetter

	req := &http.Request{
		Header: make(http.Header),
	}
	err := c.intercept(req)
	assert.Nil(t, err)

	userAgent := req.Header.Get("User-Agent")
	assert.Equal(t, uaHeader, userAgent)

	authorization := req.Header.Get("Authorization")
	assert.NotEmpty(t, authorization)

	tenant := req.Header.Get("X-Axway-Tenant-Id")
	assert.Equal(t, tenantId, tenant)

	instance := req.Header.Get("X-Axway-Instance-Id")
	assert.Equal(t, "", instance)
}

func TestResponseErrors(t *testing.T) {
	tests := []struct {
		status int
		err    error
	}{
		{status: 400, err: BadRequestError{}},
		{status: 401, err: UnauthorizedError{}},
		{status: 403, err: ForbiddenError{}},
		{status: 404, err: NotFoundError{}},
		{status: 409, err: ConflictError{}},
		{status: 500, err: InternalServerError{}},
		{status: 600, err: UnexpectedError{}},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(fmt.Sprintf("%d error", tc.status), func(t *testing.T) {
			defer gock.Off()
			gock.New("http://localhost:8080/apis").
				Get("/management/v1alpha1/environments").
				Reply(tc.status).
				JSON(mockEnv)

			_, err := client.List()

			switch tc.status {
			case 400:
				errType, ok := err.(BadRequestError)
				assert.True(t, ok)
				assert.NotEmpty(t, errType.Error())
				assert.IsType(t, BadRequestError{}, errType)
			case 401:
				errType, ok := err.(UnauthorizedError)
				assert.True(t, ok)
				assert.NotEmpty(t, errType.Error())
				assert.IsType(t, UnauthorizedError{}, errType)
			case 403:
				errType, ok := err.(ForbiddenError)
				assert.True(t, ok)
				assert.NotEmpty(t, errType.Error())
				assert.IsType(t, ForbiddenError{}, errType)
			case 404:
				errType, ok := err.(NotFoundError)
				assert.True(t, ok)
				assert.NotEmpty(t, errType.Error())
				assert.IsType(t, NotFoundError{}, errType)
			case 409:
				errType, ok := err.(ConflictError)
				assert.True(t, ok)
				assert.NotEmpty(t, errType.Error())
				assert.IsType(t, ConflictError{}, errType)
			case 500:
				errType, ok := err.(InternalServerError)
				assert.True(t, ok)
				assert.NotEmpty(t, errType.Error())
				assert.IsType(t, InternalServerError{}, errType)
			default:
				errType, ok := err.(UnexpectedError)
				assert.True(t, ok)
				assert.NotEmpty(t, errType.Error())
				assert.IsType(t, UnexpectedError{}, errType)
			}

		})
	}
}

func TestGetError(t *testing.T) {
	defer gock.Off()
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
	defer gock.Off()
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
	defer gock.Off()
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
	defer gock.Off()
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

func TestHTTPClient(t *testing.T) {
	newClient := &http.Client{}

	client := NewClient(
		"http://localhost:8080/apis",
		BasicAuth(
			"admin",
			"servicesecret",
			"admin",
			"123",
		),
		HTTPClient(newClient),
	)

	assert.Empty(t, client.userAgent)

	if newClient != client.client {
		t.Fatalf("Error: expected client.client to be %v but received %v", newClient, client.client)
	}
}

func TestUpdateMerge(t *testing.T) {
	oldAPISvc := &management.APIService{
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

	newAPISvc := &management.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Group: "management",
					Kind:  "APIService",
				},
				APIVersion: "v1alpha1",
			},
			Name: "name",
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Name: "myenv",
				},
			},
			Tags: []string{"new"},
		},
	}

	mergedTags := &management.APIService{
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

	mergeError := fmt.Errorf("merge errror")

	getError := InternalServerError{
		Errors: []apiv1.Error{{Status: 500, Detail: "Unkown error"}},
	}

	testCases := []struct {
		name             string
		replyFunc        func(*gock.Response) error
		getStatus        int
		otherStatus      int
		getResponse      interface{}
		newResource      apiv1.Interface
		mf               MergeFunc
		expectedErr      error
		expectedResource interface{}
	}{
		{
			name:        "it's a create",
			getResponse: oldAPISvc,
			newResource: newAPISvc,
			getStatus:   404,
			otherStatus: 201,
			mf: func(fetched apiv1.Interface, new apiv1.Interface) (apiv1.Interface, error) {
				return new, nil
			},
			expectedErr:      nil,
			expectedResource: newAPISvc,
		},
		{
			name:        "overwriting update",
			getResponse: oldAPISvc,
			newResource: newAPISvc,
			getStatus:   200,
			otherStatus: 200,
			mf: func(fetched apiv1.Interface, new apiv1.Interface) (apiv1.Interface, error) {
				return new, nil
			},
			expectedErr:      nil,
			expectedResource: newAPISvc,
		},
		{
			name:        "merging tags update",
			getResponse: oldAPISvc,
			newResource: newAPISvc,
			getStatus:   200,
			otherStatus: 200,
			mf: func(fetched apiv1.Interface, new apiv1.Interface) (apiv1.Interface, error) {
				f, err := fetched.AsInstance()
				if err != nil {
					return nil, err
				}

				f.SetTags(append(f.GetTags(), new.GetTags()...))

				return f, nil
			},
			expectedErr:      nil,
			expectedResource: mergedTags,
		},
		{
			name:        "merge error",
			getResponse: oldAPISvc,
			newResource: newAPISvc,
			getStatus:   200,
			otherStatus: 200,
			mf: func(fetched apiv1.Interface, new apiv1.Interface) (apiv1.Interface, error) {
				return nil, mergeError
			},
			expectedErr:      mergeError,
			expectedResource: nil,
		},
		{
			name:        "get error",
			getResponse: apiv1.ErrorResponse{Errors: getError.Errors},
			newResource: newAPISvc,
			getStatus:   500,
			otherStatus: 200,
			mf: func(fetched apiv1.Interface, new apiv1.Interface) (apiv1.Interface, error) {
				return nil, mergeError
			},
			expectedErr:      getError,
			expectedResource: nil,
		}}
	logger := WithLogger(noOpLogger{})
	c, err := NewClient("http://localhost:8080/apis", logger).ForKind(management.APIServiceGVK())

	if err != nil {
		t.Fatalf("Failed due: %s ", err)
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			defer gock.Off()
			gock.Observe(gock.DumpRequest)

			gock.New("http://localhost:8080/apis").
				Get("/management/v1alpha1/environments/myenv/apiservices/name").
				Reply(tc.getStatus).
				JSON(tc.getResponse)

			switch {
			case tc.expectedErr == mergeError:
			case tc.getStatus == 404:
				gock.New("http://localhost:8080/apis").
					Post("/management/v1alpha1/environments/myenv/apiservices").
					JSON(tc.expectedResource).
					Reply(tc.otherStatus).
					JSON(tc.expectedResource)
			case tc.getStatus == 200:
				gock.New("http://localhost:8080/apis").
					Put("/management/v1alpha1/environments/myenv/apiservices/name").
					JSON(tc.expectedResource).
					Reply(tc.otherStatus).
					JSON(tc.expectedResource)
			}

			newRI, err := tc.newResource.AsInstance()
			if err != nil {
				t.Fatal(err)
			}

			_, err = c.Update(newRI, Merge(tc.mf))

			switch {
			case err == nil && tc.expectedErr == nil:
			case err != nil && tc.expectedErr == nil:
				t.Error("Not expecting error, got: ", err)
			case err == nil && tc.expectedErr != nil:
				t.Error("Expected error: ", tc.expectedErr, "; got no error")
			case err.Error() != tc.expectedErr.Error():
				t.Error("Expected error: ", tc.expectedErr, "; got: ", err)
			}

			if gock.HasUnmatchedRequest() {
				t.Errorf("Expected more requests: %+v", gock.GetUnmatchedRequests())
			}
		})
	}
}
