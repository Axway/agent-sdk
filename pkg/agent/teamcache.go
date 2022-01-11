package agent

import (
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/jobs"
)

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

		if team.Default {
			if _, err := agent.teamMap.GetBySecondaryKey(apic.DefaultTeamKey); err != nil {
				// remove the secondary key from an existing cache item before adding it to a new one
				agent.teamMap.DeleteSecondaryKey(apic.DefaultTeamKey)
			}
			agent.teamMap.SetSecondaryKey(team.Name, apic.DefaultTeamKey)
		}
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
