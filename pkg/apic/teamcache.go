package apic

import (
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/jobs"
)

const teamMapKey = "TeamMap"

type centralTeamsCache struct {
	jobs.Job
	serviceClient *ServiceClient
}

func (j *centralTeamsCache) Ready() bool {
	return true
}

func (j *centralTeamsCache) Status() error {
	return nil
}

func (j *centralTeamsCache) Execute() error {
	platformTeams, err := j.serviceClient.getTeam(map[string]string{})
	if err != nil {
		return err
	}

	if len(platformTeams) == 0 {
		return fmt.Errorf("error: no teams returned from central")
	}

	teamMap := make(map[string]string)
	for _, team := range platformTeams {
		teamMap[team.Name] = team.ID
	}

	// cache all the teams
	return cache.GetCache().Set(teamMapKey, teamMap)
}

// registerTeamMapCacheJob -
func registerTeamMapCacheJob(serviceClient *ServiceClient) {
	job := &centralTeamsCache{
		serviceClient: serviceClient,
	}
	job.Execute()

	jobs.RegisterIntervalJobWithName(job, time.Hour, "Team Cache")
}

// GetTeamFromCache -
func GetTeamFromCache(teamName string) (string, bool) {
	obj, err := cache.GetCache().Get(teamMapKey)
	if err == nil {
		teamMap := obj.(map[string]string)
		if id, found := teamMap[teamName]; found {
			return id, found
		}
	}
	return "", false
}
