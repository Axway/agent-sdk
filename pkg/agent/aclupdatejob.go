package agent

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const envACLFormat = "%s-agent-acl"

// aclUpdateHandler - job that handles updates to the ACL in the environment
type aclUpdateJob struct {
	jobs.Job
	logger log.FieldLogger
}

func newACLUpdateJob() *aclUpdateJob {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent").
		WithComponent("aclUpdateJob")
	job := &aclUpdateJob{
		logger: logger,
	}
	return job
}

func (j *aclUpdateJob) Ready() bool {
	status, _ := hc.GetGlobalStatus()
	return status == string(hc.OK)
}

func (j *aclUpdateJob) Status() error {
	if status, _ := hc.GetGlobalStatus(); status != string(hc.OK) {
		err := fmt.Errorf("agent is marked as not running")
		j.logger.WithError(err).Trace("status failed")
		return err
	}
	return nil
}

func (j *aclUpdateJob) getACLTeamIDs(ri *v1.ResourceInstance) []string {
	teamIDs := make([]string, 0)
	acl, _ := management.NewAccessControlList("", management.EnvironmentGVK().Kind, agent.cfg.GetEnvironmentName())
	err := acl.FromInstance(ri)
	if err != nil {
		return teamIDs
	}
	for _, subject := range acl.Spec.Subjects {
		if subject.Type == v1.TeamOwner {
			teamIDs = append(teamIDs, subject.ID)
		}
	}
	teamIDs = util.RemoveDuplicateValuesFromStringSlice(teamIDs)
	sort.Strings(teamIDs)
	return teamIDs
}

func (j *aclUpdateJob) Execute() error {
	currentACL := agent.cacheManager.GetAccessControlList()
	if currentACL == nil {
		currentACL = j.getACLsFromServer()
	}

	currentTeamIDs := j.getACLTeamIDs(currentACL)
	newTeamIDs := agent.cacheManager.GetTeamsIDsInAPIServices()
	if len(newTeamIDs) == 0 {
		return nil
	}

	sort.Strings(newTeamIDs)
	if currentTeamIDs != nil && strings.Join(newTeamIDs, "") == strings.Join(currentTeamIDs, "") && !j.shouldUpdateACL(currentACL) {
		return nil
	}
	if err := j.updateACL(newTeamIDs); err != nil {
		return fmt.Errorf("acl update job failed: %s", err)
	}
	return nil
}

func (j *aclUpdateJob) shouldUpdateACL(currentACL *v1.ResourceInstance) bool {
	aclRes, _ := management.NewAccessControlList("", management.EnvironmentGVK().Kind, "")
	if err := aclRes.FromInstance(currentACL); err != nil {
		return false
	}

	// Track what we need to find
	required := map[string]bool{
		"scope":                                false,
		management.DiscoveryAgentGVK().Kind:    false,
		management.TraceabilityAgentGVK().Kind: false,
		management.ComplianceAgentGVK().Kind:   false,
	}

	// Check all rules and access entries
	for _, rule := range aclRes.Spec.Rules {
		for _, accessRule := range rule.Access {
			if accessRule.Level == "scope" {
				required["scope"] = true
			} else if accessRule.Level == "scopedKind" && accessRule.Kind != nil {
				if _, exists := required[*accessRule.Kind]; exists {
					required[*accessRule.Kind] = true
				}
			}
		}
	}

	// Check if all required elements were found
	for _, found := range required {
		if !found {
			return true
		}
	}

	return false
}

func (j aclUpdateJob) getACLsFromServer() *v1.ResourceInstance {
	emptyACL, _ := management.NewAccessControlList("", management.EnvironmentGVK().Kind, agent.cfg.GetEnvironmentName())
	acls, err := agent.apicClient.GetResources(emptyACL)
	if err != nil {
		return nil
	}

	for _, acl := range acls {
		ri, _ := acl.AsInstance()
		if ri.Name == j.getACLName() {
			agent.cacheManager.SetAccessControlList(ri)
			return ri
		}
	}
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

func (j *aclUpdateJob) createACLResource(teamIDs []string) *management.AccessControlList {
	acl, _ := management.NewAccessControlList(
		j.getACLName(),
		management.EnvironmentGVK().Kind,
		agent.cfg.GetEnvironmentName(),
	)
	acl.Spec = management.AccessControlListSpec{
		Rules: []management.AccessRules{
			{
				Access: []management.AccessLevelScope{
					{
						Level: "scope",
					},
					{
						Level: "scopedKind",
						Kind:  Ptr(management.DiscoveryAgentGVK().Kind),
					},
					{
						Level: "scopedKind",
						Kind:  Ptr(management.TraceabilityAgentGVK().Kind),
					},
					{
						Level: "scopedKind",
						Kind:  Ptr(management.ComplianceAgentGVK().Kind),
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
	j.logger.Trace("acl about to be updated")
	acl := j.createACLResource(teamIDs)
	if currentACL != nil {
		acl, err = agent.apicClient.UpdateAccessControlList(acl)
	} else {
		acl, err = agent.apicClient.CreateAccessControlList(acl)
	}

	if err == nil {
		aclInstance, err := acl.AsInstance()
		if err == nil {
			agent.cacheManager.SetAccessControlList(aclInstance)
		}
	}

	return err
}

// registerAccessControlListHandler -
func registerAccessControlListHandler() {
	job := newACLUpdateJob()

	jobs.RegisterIntervalJobWithName(job, agent.cfg.GetPollInterval(), "Access Control List")
}
