package agent

import (
	"sync"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	status = "status"
	state  = "state"
)

type cacheManager interface {
	GetWatchResourceCacheKeys(group, kind string) []string
	GetWatchResourceByKey(key string) *v1.ResourceInstance
}

type apicClient interface {
	UpdateResourceInstance(ri v1.Interface) (*v1.ResourceInstance, error)
	CreateSubResource(rm v1.ResourceMeta, subs map[string]interface{}) error
}

type credentialValidator struct {
	jobs.Job
	id           string
	logger       log.FieldLogger
	cacheManager cacheManager
	client       apicClient
}

func newCredentialChecker(cacheManager cacheManager, client apicClient) *credentialValidator {
	return &credentialValidator{
		logger:       log.NewFieldLogger().WithComponent("credentialValidator"),
		cacheManager: cacheManager,
		client:       client,
	}
}

// Ready -
func (j *credentialValidator) Ready() bool {
	return true
}

// Status -
func (j *credentialValidator) Status() error {
	return nil
}

// Execute -
func (j *credentialValidator) Execute() error {
	j.logger.Debug("validating credentials for expiration")

	if agent.cfg.GetCredentialConfig() == nil ||
		!agent.cfg.GetCredentialConfig().ShouldDeprovisionExpired() {
		return nil
	}

	// Get all of the credentials from the cache
	credKeys := j.cacheManager.GetWatchResourceCacheKeys(management.CredentialGVK().Group, management.CredentialGVK().Kind)

	// loop all the keys in the cache and check if any have expired
	now := time.Now()
	wg := &sync.WaitGroup{}
	for _, k := range credKeys {
		wg.Add(1)
		func(credKey string) {
			j.validateCredential(credKey, now)
		}(k)
	}

	return nil
}

func (j *credentialValidator) validateCredential(credKey string, now time.Time) {
	logger := j.logger.WithField("cacheKey", credKey)
	res := j.cacheManager.GetWatchResourceByKey(credKey)
	if res == nil {
		logger.Error("could not get resource by key")
		return
	}

	cred := &management.Credential{}
	err := cred.FromInstance(res)
	if err != nil {
		logger.WithError(err).Error("could not convert resource instance to credential")
		return
	}

	if cred.Policies.Expiry == nil {
		return
	}

	expTime := time.Time(cred.Policies.Expiry.Timestamp)
	if expTime.IsZero() {
		// cred does not expire
		return
	}

	logger = logger.WithField("credName", cred.Name).WithField("expiration", expTime.Format(v1.APIServerTimeFormat))
	logger.Trace("validating credential")

	if expTime.Before(now) {
		logger.Info("Credential has expired, updating Central")
		cred.Status.Level = provisioning.Pending.String()

		// update state so the inactivated credential will come back for removal
		cred.Spec.State = management.CredentialSpecState{
			Name:   v1.Inactive,
			Reason: provisioning.CredExpDetail,
		}

		// only update a subset of the sub resources
		subResources := map[string]interface{}{
			status: cred.Status,
			state:  cred.State,
		}

		_, err = j.client.UpdateResourceInstance(cred)
		if err != nil {
			logger.WithError(err).Error("error update credential resources")
		}

		err = j.client.CreateSubResource(cred.ResourceMeta, subResources)
		if err != nil {
			logger.WithError(err).Error("error creating subresources")
		}
	}
}

func registerCredentialChecker() *credentialValidator {
	if agent.cfg.GetAgentType() != config.DiscoveryAgent {
		return nil
	}

	c := newCredentialChecker(agent.cacheManager, agent.apicClient)

	err := agent.cfg.SetWatchResourceFilters([]config.ResourceFilter{
		{
			Group:            management.CredentialGVK().Group,
			Kind:             management.CredentialGVK().Kind,
			Name:             "*",
			IsCachedResource: true,
		},
	})
	if err != nil {
		c.logger.WithError(err).Error("could not watch for the credential resource in the credential validator job")
		return nil
	}

	id, err := jobs.RegisterScheduledJobWithName(c, "@hourly", "CredentialValidator")
	if err != nil {
		c.logger.WithError(err).Error("could not start the credential validator job")
		return nil
	}
	c.logger.Debug("registered the credential validator")
	c.id = id

	return c
}
