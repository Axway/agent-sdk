package agent

import (
	"fmt"
	"sort"
	"sync"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const envACLFormat = "%s-agent-acl"

var waitForTime = time.Minute

//aclUpdateHandler - job that handles updates to the ACL in the environment
type aclUpdateHandlerJob struct {
	jobs.Job
	currentACL       *v1alpha1.AccessControlList
	existingTeamIDs  []string
	newTeamIDs       []string
	stopChan         chan interface{}
	teamChan         chan string
	runningChan      chan bool
	isRunning        bool
	countdownStarted bool // signal that the countdown to push updated teams has started
	newTeamMutex     sync.Mutex
	countdownMutex   sync.Mutex
}

func newACLUpdateHandlerJob(teamChanel chan string) *aclUpdateHandlerJob {
	job := &aclUpdateHandlerJob{
		existingTeamIDs: make([]string, 0),
		newTeamIDs:      make([]string, 0),
		stopChan:        make(chan interface{}),
		teamChan:        teamChanel,
		runningChan:     make(chan bool),
		isRunning:       false,
		newTeamMutex:    sync.Mutex{},
		countdownMutex:  sync.Mutex{},
	}
	go job.statusUpdate()
	return job
}

func (j *aclUpdateHandlerJob) Ready() bool {
	status := hc.GetStatus(healthcheckEndpoint)
	return status == hc.OK
}

func (j *aclUpdateHandlerJob) Status() error {
	if !j.isRunning {
		return fmt.Errorf("acl update handler not running")
	}

	status := hc.GetStatus(healthcheckEndpoint)
	if status == hc.OK {
		return nil
	}
	return fmt.Errorf("could not establish a connection to APIC to update the acl")
}

func (j *aclUpdateHandlerJob) Execute() error {
	j.started()
	defer j.stopped()

	j.initializeACLJob()

	for {
		select {
		case teamID, ok := <-j.teamChan:
			if !ok {
				err := fmt.Errorf("team channel was closed")
				return err
			}
			j.handleTeam(teamID)
		case <-j.stopChan:
			log.Info("Stopping the api handler")
			return nil
		}
	}
}

func (j *aclUpdateHandlerJob) getACLName() string {
	return fmt.Sprintf(envACLFormat, GetCentralConfig().GetEnvironmentName())
}

func (j *aclUpdateHandlerJob) statusUpdate() {
	for {
		select {
		case update := <-j.runningChan:
			j.isRunning = update
		}
	}
}

func (j *aclUpdateHandlerJob) initializeACLJob() {
	acl, err := agent.apicClient.GetAccessControlList(j.getACLName())
	if err != nil {
		go j.updateACL()
		return
	}

	j.currentACL = acl
	for _, subject := range acl.Spec.Subjects {
		if subject.Type == v1.TeamOwner {
			j.existingTeamIDs = append(j.existingTeamIDs, subject.ID)
		}
	}

	numTeamsOnEnv := len(j.existingTeamIDs)
	j.existingTeamIDs = util.RemoveDuplicateValuesFromStringSlice(j.existingTeamIDs)
	if len(j.existingTeamIDs) != numTeamsOnEnv {
		go j.updateACL()
		return
	}
	sort.Strings(j.existingTeamIDs)
}

func (j *aclUpdateHandlerJob) started() {
	j.runningChan <- true
}

func (j *aclUpdateHandlerJob) stopped() {
	j.runningChan <- false
}

func (j *aclUpdateHandlerJob) handleTeam(teamID string) {
	log.Tracef("acl update job received team id %s", teamID)
	// lock so an update does not happen until the team is added to the array
	j.newTeamMutex.Lock()
	defer j.newTeamMutex.Unlock()

	if util.IsItemInSlice(j.existingTeamIDs, teamID) {
		log.Tracef("team id %s already in acl", teamID)
		return
	}

	if util.IsItemInSlice(j.newTeamIDs, teamID) {
		log.Tracef("team id %s already in acl update process", teamID)
		return
	}

	j.newTeamIDs = append(j.newTeamIDs, teamID)
	go j.updateACL()
}

func (j *aclUpdateHandlerJob) createACLResource(teamIDs []string) *v1alpha1.AccessControlList {
	acl := j.currentACL
	if acl == nil {
		acl = &v1alpha1.AccessControlList{
			ResourceMeta: v1.ResourceMeta{
				GroupVersionKind: v1alpha1.AccessControlListGVK(),
				Name:             j.getACLName(),
				Title:            j.getACLName(),
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

func (j *aclUpdateHandlerJob) updateACL() {
	j.countdownMutex.Lock()
	if j.countdownStarted {
		j.countdownMutex.Unlock()
		return
	}

	j.countdownStarted = true
	j.countdownMutex.Unlock()
	log.Tracef("waiting %s, for more teams, before updating acl", waitForTime)
	time.Sleep(waitForTime)

	// lock so teams are not added to the array until the update is done
	j.newTeamMutex.Lock()
	defer func() {
		j.countdownMutex.Lock()
		// reset this signal once function completes
		j.countdownStarted = false
		j.newTeamMutex.Unlock()
		j.countdownMutex.Unlock()
	}()

	var err error
	var acl *v1alpha1.AccessControlList
	log.Tracef("acl about to be updated")
	if j.currentACL != nil {
		acl, err = agent.apicClient.UpdateAccessControlList(j.createACLResource(append(j.existingTeamIDs, j.newTeamIDs...)))
	} else {
		acl, err = agent.apicClient.CreateAccessControlList(j.createACLResource(j.newTeamIDs))
	}

	if err == nil {
		log.Tracef("acl has been updated")
		j.existingTeamIDs = append(j.existingTeamIDs, j.newTeamIDs...)
		j.newTeamIDs = make([]string, 0)
		sort.Strings(j.existingTeamIDs)
		j.currentACL = acl
	} else {
		log.Errorf("error in acl handler: %s", err)
	}
}

// registerAccessControlListHandler -
func registerAccessControlListHandler(teamChannel chan string) {
	job := newACLUpdateHandlerJob(teamChannel)

	jobs.RegisterChannelJobWithName(job, job.stopChan, "Access Control List Handler")
}
