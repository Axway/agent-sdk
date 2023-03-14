package agent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func generateTestACL(name string, teams []string) *management.AccessControlList {
	acl := &management.AccessControlList{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: management.AccessControlListGVK(),
			Name:             name,
			Title:            name,
		},
		Spec: management.AccessControlListSpec{
			Rules: []management.AccessRules{
				{
					Access: []management.AccessLevelScope{
						{
							Level: "scope",
						},
					},
				},
			},
		},
	}

	// Add all the teams
	acl.Spec.Subjects = make([]v1.Owner, 0)
	for _, id := range teams {
		acl.Spec.Subjects = append(acl.Spec.Subjects, v1.Owner{
			Type: v1.TeamOwner,
			ID:   id,
		})
	}
	return acl
}

func TestACLUpdateHandlerJob(t *testing.T) {
	testCases := []struct {
		name      string
		envName   string
		teamSets  [][]string
		aclCached []bool
	}{
		{
			name:    "No ACL-One Team",
			envName: "Environment",
			teamSets: [][]string{
				{},
				{"TeamA"},
				{},
			},
			aclCached: []bool{false, true, true},
		},
		{
			name:    "Existing ACL-Team Known",
			envName: "Environment",
			teamSets: [][]string{
				{"TeamA"},
				{"TeamA"},
				{},
			},
			aclCached: []bool{true, true, true},
		},
		{
			name:    "Existing ACL-Only Init Teams",
			envName: "Environment",
			teamSets: [][]string{
				{"TeamA"},
				{"TeamA", "TeamB", "TeamC"},
				{},
			},
			aclCached: []bool{true, true, true},
		},
		{
			name:    "No ACL-Same Team 3x",
			envName: "Environment",
			teamSets: [][]string{
				{},
				{"TeamA", "TeamA", "TeamA"},
				{},
			},
			aclCached: []bool{false, true, true},
		},
		{
			name:    "No ACL-Init Teams-New Teams",
			envName: "Environment",
			teamSets: [][]string{
				{},
				{"TeamA"},
				{"TeamB", "TeamC"},
			},
			aclCached: []bool{false, true, true},
		},
		{
			name:    "Existing ACL-Init Teams-New Teams",
			envName: "Environment",
			teamSets: [][]string{
				{"TeamA"},
				{"TeamA", "TeamB"},
				{"TeamC", "TeamD"},
			},
			aclCached: []bool{true, true, true},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// initialize the http responses
			s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				if strings.Contains(req.RequestURI, "/auth") {
					token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
					resp.Write([]byte(token))
				}
				if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/"+test.envName+"/accesscontrollists") {
					aclReturn, _ := io.ReadAll(req.Body)
					resp.WriteHeader(http.StatusCreated)
					resp.Write(aclReturn)
				}
			}))
			defer s.Close()

			cfg := createCentralCfg(s.URL, test.envName)
			resetResources()
			err := Initialize(cfg)
			assert.Nil(t, err)

			// create the job to test
			job := newACLUpdateJob()

			combinedTeams := make([]string, 0)
			for i, teamList := range test.teamSets {
				combinedTeams = append(combinedTeams, teamList...)
				combinedTeams = util.RemoveDuplicateValuesFromStringSlice(combinedTeams)
				expectedACL := generateTestACL(job.getACLName(), combinedTeams)

				// load api services with the teams
				for _, team := range combinedTeams {
					agent.cacheManager.AddAPIService(&v1.ResourceInstance{
						ResourceMeta: v1.ResourceMeta{
							GroupVersionKind: management.APIServiceGVK(),
							Name:             team,
							Title:            team,
							SubResources: map[string]interface{}{
								definitions.XAgentDetails: map[string]interface{}{
									definitions.AttrExternalAPIID: team,
								},
							},
						},
						Owner: &v1.Owner{
							Type: v1.TeamOwner,
							ID:   team,
						},
					})
				}

				// adjust the wait time and start the job
				job.Execute()

				// acl from cache
				if test.aclCached[i] {
					var acl management.AccessControlList
					cachedACL := agent.cacheManager.GetAccessControlList()
					acl.FromInstance(cachedACL)
					assert.Equal(t, len(expectedACL.Spec.Subjects), len(acl.Spec.Subjects))
				} else {
					assert.Nil(t, agent.cacheManager.GetAccessControlList())
				}
			}
		})
	}
}

func TestInitializeACLJob(t *testing.T) {
	tests := []struct {
		name      string
		loadCache bool
		returnACL bool
		apiCalled bool
	}{
		{
			name:      "ACL not in cache or Central",
			loadCache: false,
			returnACL: false,
			apiCalled: true,
		},
		{
			name:      "ACL in cache",
			loadCache: true,
			returnACL: false,
			apiCalled: false,
		},
		{
			name:      "ACL not in cache, on Central",
			loadCache: false,
			returnACL: true,
			apiCalled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var expectedACL *management.AccessControlList
			var apiCalled bool
			// initialize the http responses
			s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				if strings.Contains(req.RequestURI, "/auth") {
					token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
					resp.Write([]byte(token))
				}
				if strings.Contains(req.RequestURI, "/accesscontrollists") {
					switch {
					case tt.returnACL:
						resp.WriteHeader(http.StatusOK)
						data, _ := json.Marshal(expectedACL)
						resp.Write(data)
					default:
						resp.WriteHeader(http.StatusNotFound)
					}
					apiCalled = true
				}
			}))
			defer s.Close()

			cfg := createCentralCfg(s.URL, "environment")
			resetResources()
			err := Initialize(cfg)
			assert.Nil(t, err)

			job := newACLUpdateJob()
			expectedACL = generateTestACL(job.getACLName(), []string{"TeamA", "TeamB"})

			if tt.loadCache {
				aclInstance, _ := expectedACL.AsInstance()
				agent.cacheManager.SetAccessControlList(aclInstance)
			}

			job.initializeACLJob()

			cachedACL := agent.cacheManager.GetAccessControlList()
			if tt.loadCache || tt.returnACL {
				assert.NotNil(t, cachedACL)
				var acl management.AccessControlList
				acl.FromInstance(cachedACL)
				assert.True(t, assert.ObjectsAreEqualValues(expectedACL.Spec.Subjects, acl.Spec.Subjects))
			} else {
				assert.Nil(t, cachedACL)
			}
			assert.Equal(t, tt.apiCalled, apiCalled)
		})
	}
}
