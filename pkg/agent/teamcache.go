package agent

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// QA EnvVars
const qaTeamCacheInterval = "QA_CENTRAL_TEAMCACHE_INTERVAL"

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
		savedTeam := agent.cacheManager.GetTeamByID(team.ID)
		if savedTeam == nil {
			var params = []interface{}{&team}
			pointerIdx := firstPointerIdx(params)

			if pointerIdx > -1 {
				log.Debug("Shane - Contains pointer ", pointerIdx)
			}
			agent.cacheManager.AddTeam(team)
		}
	}
	return nil
}

func firstPointerIdx(s []interface{}) int {
	for i, v := range s {
		if reflect.ValueOf(v).Kind() == reflect.Ptr {
			log.Debug("Shane - Contains pointer value of ", reflect.ValueOf(v))
			return i
		}
	}
	return -1
}

// registerTeamMapCacheJob -
func registerTeamMapCacheJob() {
	job := &centralTeamsCache{}

	// execute the job on startup to populate the team cache
	job.Execute()
	jobs.RegisterIntervalJobWithName(job, getJobInterval(), "Team Cache")
}

func getJobInterval() time.Duration {
	interval := time.Hour
	// check for QA env vars
	if val := os.Getenv(qaTeamCacheInterval); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			log.Tracef("Using %s (%s) rather than the default (%s) for non-QA", qaTeamCacheInterval, val, time.Hour)
			interval = duration
		} else {
			log.Tracef("Could not use %s (%s) it is not a proper duration", qaTeamCacheInterval, val)
		}
	}

	return interval
}

// GetTeamByName - Returns the PlatformTeam associated with the name
func GetTeamByName(name string) *definitions.PlatformTeam {
	if agent.cacheManager != nil {
		return agent.cacheManager.GetTeamByName(name)
	}
	return nil
}

// GetTeamByID - Returns the PlatformTeam associated with the id
func GetTeamByID(id string) *definitions.PlatformTeam {
	if agent.cacheManager != nil {
		return agent.cacheManager.GetTeamByID(id)
	}
	return nil
}
