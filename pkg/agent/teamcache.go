package agent

import (
	"fmt"
	"os"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// QA EnvVars
const qaTeamCacheInterval = "QA_CENTRAL_TEAMCACHE_INTERVAL"

type centralTeamsCache struct {
	jobs.Job
	teamChannel chan string
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
		if id, err := agent.teamMap.Get(team.Name); err != nil || id != team.ID {
			err = agent.teamMap.Set(team.Name, team.ID)
			if j.teamChannel != nil {
				log.Tracef("sending %s (%s) team to acl", team.Name, team.ID)
				j.teamChannel <- team.ID
			}
		}
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
func registerTeamMapCacheJob(teamChannel chan string) {
	job := &centralTeamsCache{
		teamChannel: teamChannel,
	}

	// execute the job on startup to populate the team cache
	job.Execute()

	interval := time.Hour
	// chgeck for QA env vars
	if val := os.Getenv(qaTeamCacheInterval); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			log.Tracef("Using %s (%s) rather than the default (%s) for non-QA", qaTeamCacheInterval, val, time.Hour)
			interval = duration
		} else {
			log.Tracef("Could not use %s (%s) it is not a proper duration", qaTeamCacheInterval, val)
		}
	}

	jobs.RegisterIntervalJobWithName(job, interval, "Team Cache")
}

// GetTeamFromCache -
func GetTeamFromCache(teamName string) (string, bool) {
	id, found := agent.teamMap.Get(teamName)
	if teamName == "" {
		// get the default team
		id, found = agent.teamMap.GetBySecondaryKey(apic.DefaultTeamKey)
	}
	if found != nil {
		return "", false
	}
	return id.(string), true
}
