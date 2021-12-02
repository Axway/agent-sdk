package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestTeamCache(t *testing.T) {
	type queryRes struct {
		teams     []string
		numOnChan int
	}

	testCases := []struct {
		name       string
		queries    []queryRes
		notInCache []string
	}{
		{
			name: "Teams",
			queries: []queryRes{
				{
					teams:     []string{"TeamA"},
					numOnChan: 1,
				},
				{
					teams:     []string{"TeamA", "TeamB", "TeamC"},
					numOnChan: 2,
				},
				{
					teams:     []string{"TeamA", "TeamB", "TeamC", "TeamD"},
					numOnChan: 1,
				},
			},
			notInCache: []string{"TeamE"},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			request := 0
			s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				switch {
				case strings.Contains(req.RequestURI, "/auth"):
					token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
					resp.Write([]byte(token))
				case strings.Contains(req.RequestURI, "platformTeams"):
					// add teams to reply
					reply := make([]*apic.PlatformTeam, 0)
					for i, team := range test.queries[request].teams {
						reply = append(reply, &apic.PlatformTeam{
							ID:      team,
							Name:    team,
							Default: i == 0,
						})
					}
					request++
					data, _ := json.Marshal(reply)
					resp.Write(data)
				}
			}))
			defer s.Close()

			cfg := createCentralCfg(s.URL, "env")
			resetResources()
			agent.teamMap = nil
			err := Initialize(cfg)
			assert.Nil(t, err)
			assert.NotNil(t, agent)
			assert.NotNil(t, agent.apicClient)

			teamChanel := make(chan string)
			job := centralTeamsCache{teamChannel: teamChanel}

			// receive all the teams
			allTeams := make([]string, 0)
			receivedTeams := make([]string, 0)
			for _, q := range test.queries {
				allTeams = append(allTeams, q.teams...)
				go job.Execute()
				expected := q.numOnChan
				for expected > 0 {
					team := <-teamChanel
					receivedTeams = append(receivedTeams, team)
					expected--
				}
			}
			expectedTeams := util.RemoveDuplicateValuesFromStringSlice(allTeams)

			// test validations
			assert.ElementsMatch(t, expectedTeams, receivedTeams)
			defTeam, found := GetTeamFromCache("")
			assert.True(t, found, "a default team was not in the cache")
			assert.Equal(t, expectedTeams[0], defTeam)
			for _, team := range expectedTeams {
				_, found := GetTeamFromCache(team)
				assert.True(t, found, "%s team was not in the cache", team)
			}
			for _, team := range test.notInCache {
				_, found := GetTeamFromCache(team)
				assert.False(t, found, "%s team was in the cache", team)
			}
		})
	}
}
