package agent

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func generateTestACL(name string, teams []string) *v1alpha1.AccessControlList {
	acl := &v1alpha1.AccessControlList{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.AccessControlListGVK(),
			Name:             name,
			Title:            name,
		},
		Spec: v1alpha1.AccessControlListSpec{
			Rules: []v1alpha1.AccessRules{
				{
					Access: []v1alpha1.AccessLevelScope{
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
		aclExists bool
		aclTeams  []string
		initTeams []string
		newTeams  []string
		expCalls  map[string]int
	}{
		{
			name:      "No ACL-One Team",
			envName:   "Environment",
			aclExists: false,
			aclTeams:  []string{},
			initTeams: []string{"TeamA"},
			newTeams:  []string{},
			expCalls: map[string]int{
				http.MethodGet:  1,
				http.MethodPost: 1,
				http.MethodPut:  0,
			},
		},
		{
			name:      "Existing ACL-Team Known",
			envName:   "Environment",
			aclExists: true,
			aclTeams:  []string{"TeamA"},
			initTeams: []string{"TeamA"},
			newTeams:  []string{},
			expCalls: map[string]int{
				http.MethodGet:  1,
				http.MethodPost: 0,
				http.MethodPut:  0,
			},
		},
		{
			name:      "Existing ACL-Only Init Teams",
			envName:   "Environment",
			aclExists: true,
			aclTeams:  []string{"TeamA"},
			initTeams: []string{"TeamA", "TeamB", "TeamC"},
			newTeams:  []string{},
			expCalls: map[string]int{
				http.MethodGet:  1,
				http.MethodPost: 0,
				http.MethodPut:  1,
			},
		},
		{
			name:      "No ACL-Same Team 3x",
			envName:   "Environment",
			aclExists: false,
			aclTeams:  []string{},
			initTeams: []string{"TeamA", "TeamA", "TeamA"},
			newTeams:  []string{},
			expCalls: map[string]int{
				http.MethodGet:  1,
				http.MethodPost: 1,
				http.MethodPut:  0,
			},
		},
		{
			name:      "No ACL-Init Teams-New Teams",
			envName:   "Environment",
			aclExists: false,
			aclTeams:  []string{},
			initTeams: []string{"TeamA"},
			newTeams:  []string{"TeamB", "TeamC"},
			expCalls: map[string]int{
				http.MethodGet:  1,
				http.MethodPost: 1,
				http.MethodPut:  1,
			},
		},
		{
			name:      "Existing ACL-Init Teams-New Teams",
			envName:   "Environment",
			aclExists: true,
			aclTeams:  []string{"TeamA"},
			initTeams: []string{"TeamA", "TeamB"},
			newTeams:  []string{"TeamC", "TeamD"},
			expCalls: map[string]int{
				http.MethodGet:  1,
				http.MethodPost: 0,
				http.MethodPut:  2,
			},
		},
	}

	teamChannel := make(chan string)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			numCalls := map[string]int{
				http.MethodGet:  0,
				http.MethodPost: 0,
				http.MethodPut:  0,
			}
			// initial acl
			aclFromEnv := generateTestACL(test.envName, test.aclTeams)

			// acl post, when no init or put when init
			allTeams := util.RemoveDuplicateValuesFromStringSlice(append(test.aclTeams, test.initTeams...))
			sort.Strings(allTeams)
			aclAfterInit := generateTestACL(test.envName, allTeams)

			// acl put when new teams added
			finalTeams := util.RemoveDuplicateValuesFromStringSlice(append(allTeams, test.newTeams...))
			sort.Strings(finalTeams)
			aclFinal := generateTestACL(test.envName, finalTeams)

			// initialize the http responses
			s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				if strings.Contains(req.RequestURI, "/auth") {
					token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
					resp.Write([]byte(token))
				}
				if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/"+test.envName+"/accesscontrollists") {
					numCalls[req.Method]++
					aclReturn := make([]byte, 0)
					aclReturn, _ = ioutil.ReadAll(req.Body)
					switch {
					case req.Method == http.MethodGet && test.aclExists:
						resp.WriteHeader(http.StatusOK)
						aclReturn, _ = json.Marshal(aclFromEnv)
					case req.Method == http.MethodGet && !test.aclExists:
						resp.WriteHeader(http.StatusNotFound)
						data, _ := json.Marshal(&apic.ResponseError{
							Errors: []apic.APIError{
								{
									Status: 404,
									Title:  "Error",
									Detail: "Error",
								},
							},
						})
						resp.Write(data)
						return
					case req.Method == http.MethodPost:
						resp.WriteHeader(http.StatusCreated)
					case req.Method == http.MethodPut && test.aclExists && numCalls[http.MethodPut] == 1:
						resp.WriteHeader(http.StatusOK)
					case req.Method == http.MethodPut:
						resp.WriteHeader(http.StatusOK)
					}

					resp.Write(aclReturn)
				}
			}))
			defer s.Close()

			cfg := createCentralCfg(s.URL, test.envName)
			resetResources()
			err := Initialize(cfg)
			assert.Nil(t, err)

			// create the job to test
			job := newACLUpdateHandlerJob(teamChannel)
			origWaitForTime := waitForTime

			defer func() {
				// clean up the job
				waitForTime = origWaitForTime
				job.stopChan <- nil
			}()

			// adjust the wait time and start the job
			waitForTime = 5 * time.Millisecond
			go job.Execute()

			// validate that existing teams were collected
			if test.aclExists {
				time.Sleep(waitForTime * 2)
				sort.Strings(test.aclTeams)
				assert.Exactly(t, test.aclTeams, job.existingTeamIDs, "existing ACL did not have expected team ids")
				assert.Exactly(t, aclFromEnv.Spec, job.currentACL.Spec, "existing ACL did not match what the job stored")
			}

			// send the initTeams
			for _, id := range test.initTeams {
				teamChannel <- id
			}

			// wait for the waitTime
			time.Sleep(waitForTime * 2)

			// validate new teams were added to the acl
			if test.aclExists {
				assert.Exactly(t, allTeams, job.existingTeamIDs, "new ACL on init did not have expected team ids")
				assert.Exactly(t, aclAfterInit.Spec, job.currentACL.Spec, "new ACL on init did not match what the job stored")
			}

			// wait for the waitTime
			time.Sleep(waitForTime * 2)

			// send the newTeams
			for _, id := range test.newTeams {
				teamChannel <- id
			}

			time.Sleep(waitForTime * 2)

			// validate new teams were added to the acl
			if test.aclExists {
				assert.Exactly(t, finalTeams, job.existingTeamIDs, "final ACL did not have expected team ids")
				assert.Exactly(t, aclFinal.Spec, job.currentACL.Spec, "final ACL did not match what the job stored")
			}

			// wait for the waitTime
			time.Sleep(waitForTime * 2)

			// validate api calls
			for method, count := range test.expCalls {
				assert.Equalf(t, count, numCalls[method], "Incorrect number of %s calls made by acl handler", method)
			}
		})
	}
}
