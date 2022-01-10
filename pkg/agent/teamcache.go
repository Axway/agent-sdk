package agent

import (
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"
)

const teamMapKey = "TeamMap"

type centralTeamsCache struct {
	jobs.Job
}

func (j *centralTeamsCache) Ready() bool {
	return true
}

func (j *centralTeamsCache) Status() error {
	return nil
}

func (j *centralTeamsCache) Execute() error {
	platformTeams, err := agent.apicClient.GetTeam(map[string]string{})
	if err != nil {
		return err
	}

	if len(platformTeams) == 0 {
		return fmt.Errorf("error: no teams returned from central")
	}

	for _, team := range platformTeams {
		err = agent.teamMap.Set(team.Name, team.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// registerTeamMapCacheJob -
func registerTeamMapCacheJob() {
	job := &centralTeamsCache{}

	jobs.RegisterIntervalJobWithName(job, time.Hour, "Team Cache")
}

// GetTeamFromCache -
func GetTeamFromCache(teamName string) (string, bool) {
	id, found := agent.teamMap.Get(teamName)
	if found != nil {
		return "", false
	}
	return id.(string), true
}
