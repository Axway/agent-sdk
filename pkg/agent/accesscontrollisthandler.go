package agent

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const envACLFormat = "%s-agent-acl"

// aclUpdateHandler - job that handles updates to the ACL in the environment
type aclUpdateJob struct {
	jobs.Job
	lastTeamIDs []string
}

func newACLUpdateJob() *aclUpdateJob {
	job := &aclUpdateJob{}
	return job
}

func (j *aclUpdateJob) Ready() bool {
	status := hc.GetStatus(healthcheckEndpoint)
	ready := status == hc.OK
	if ready {
		j.initializeACLJob()
	}
	return ready
}

func (j *aclUpdateJob) Status() error {
	status := hc.GetStatus(healthcheckEndpoint)
	if status == hc.OK {
		return nil
	}
	return fmt.Errorf("could not establish a connection to APIC to update the acl")
}

func (j *aclUpdateJob) Execute() error {
	newTeamIDs := agent.cacheManager.GetTeamsIDsInAPIServices()
	newTeamIDs = sort.StringSlice(newTeamIDs)
	if j.lastTeamIDs != nil && strings.Join(newTeamIDs, "") == strings.Join(j.lastTeamIDs, "") {
		return nil
	}
	if err := j.updateACL(newTeamIDs); err != nil {
		return err
	}
	j.lastTeamIDs = sort.StringSlice(newTeamIDs)
	return nil
}

func (j *aclUpdateJob) getACLName() string {
	return fmt.Sprintf(envACLFormat, GetCentralConfig().GetEnvironmentName())
}

func (j *aclUpdateJob) initializeACLJob() {
	if acl := agent.cacheManager.GetAccessControlList(); acl != nil {
		return
	}

	acl, err := agent.apicClient.GetAccessControlList(j.getACLName())
	if err != nil {
		return
	}

	if aclInstance, err := acl.AsInstance(); err == nil {
		agent.cacheManager.SetAccessControlList(aclInstance)
	}
}

func (j *aclUpdateJob) createACLResource(teamIDs []string) *v1alpha1.AccessControlList {
	acl, _ := v1alpha1.NewAccessControlList(
		j.getACLName(),
		v1alpha1.EnvironmentGVK().Kind,
		agent.cfg.GetEnvironmentName(),
	)
	acl.Spec = v1alpha1.AccessControlListSpec{
		Rules: []v1alpha1.AccessRules{
			{
				Access: []v1alpha1.AccessLevelScope{
					{
						Level: "scope",
					},
				},
			},
		},
	}

	// Add all the teams
	acl.Spec.Subjects = make([]v1.Owner, 0)
	for _, id := range teamIDs {
		acl.Spec.Subjects = append(acl.Spec.Subjects, v1.Owner{
			Type: v1.TeamOwner,
			ID:   id,
		})
	}

	return acl
}

func (j *aclUpdateJob) updateACL(teamIDs []string) error {
	// do not add an acl if there are no teamIDs and an ACL currently does not exist
	currentACL := agent.cacheManager.GetAccessControlList()
	if len(teamIDs) == 0 && currentACL == nil {
		return nil
	}

	var err error
	log.Tracef("acl about to be updated")
	acl := j.createACLResource(teamIDs)
	if currentACL != nil {
		acl, err = agent.apicClient.UpdateAccessControlList(acl)
	} else {
		acl, err = agent.apicClient.CreateAccessControlList(acl)
	}

	return err
}

// registerAccessControlListHandler -
func registerAccessControlListHandler() {
	job := newACLUpdateJob()

	jobs.RegisterIntervalJobWithName(job, agent.cfg.GetPollInterval(), "Access Control List")
}
