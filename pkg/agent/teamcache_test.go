package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/stretchr/testify/assert"
)

func TestTeamCache(t *testing.T) {
	testCases := map[string]struct {
		skip                 bool
		teams                []*definitions.PlatformTeam
		cached               []*definitions.PlatformTeam
		expectedTeamsInCache int
	}{
		"Should save one team to the cache": {
			skip:                 false,
			expectedTeamsInCache: 1,
			cached:               []*definitions.PlatformTeam{},
			teams: []*definitions.PlatformTeam{
				{
					Name:    "TeamA",
					ID:      "1",
					Default: true,
				},
			},
		},
		"Should save two teams to the cache, and remove a team that was added": {
			skip:                 false,
			expectedTeamsInCache: 2,
			cached: []*definitions.PlatformTeam{
				{
					Name:    "TeamA",
					ID:      "1",
					Default: true,
				},
			},
			teams: []*definitions.PlatformTeam{
				{
					Name:    "TeamB",
					ID:      "2",
					Default: false,
				},
				{
					Name:    "TeamC",
					ID:      "3",
					Default: false,
				},
			},
		},
		"Should save 4 teams in the cache": {
			skip:                 false,
			expectedTeamsInCache: 4,
			cached: []*definitions.PlatformTeam{
				{
					Name:    "TeamA",
					ID:      "1",
					Default: true,
				},
				{
					Name:    "TeamB",
					ID:      "2",
					Default: false,
				},
				{
					Name:    "TeamC",
					ID:      "3",
					Default: false,
				},
			},
			teams: []*definitions.PlatformTeam{
				{
					Name:    "TeamA",
					ID:      "1",
					Default: true,
				},
				{
					Name:    "TeamB",
					ID:      "2",
					Default: false,
				},
				{
					Name:    "TeamC",
					ID:      "3",
					Default: false,
				},
				{
					Name:    "TeamD",
					ID:      "4",
					Default: false,
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			if test.skip {
				t.Skip("Skipping test")
			}
			s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				switch {
				case strings.Contains(req.RequestURI, "/auth"):
					token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
					resp.Write([]byte(token))
				case strings.Contains(req.RequestURI, "platformTeams"):
					data, _ := json.Marshal(test.teams)
					resp.Write(data)
				}
			}))
			defer s.Close()

			cfg := createCentralCfg(s.URL, "env")
			caches := cache.NewAgentCacheManager(cfg, false)

			for _, item := range test.cached {
				caches.AddTeam(item)
			}

			resetResources()
			agent.teamMap = nil
			err := Initialize(cfg)
			assert.Nil(t, err)
			assert.NotNil(t, agent.apicClient)

			handler.RefreshTeamCache(agent.apicClient, agent.cacheManager)
			teams := agent.cacheManager.GetTeamCache().GetKeys()
			assert.Equal(t, test.expectedTeamsInCache, len(teams))
		})
	}
}
