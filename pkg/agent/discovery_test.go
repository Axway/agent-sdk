package agent

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func setupDiscoveryTest(t *testing.T) func() {
	t.Helper()
	agent.cfg = createCentralCfg("http://test", "testEnv")
	agent.publishingLockAcquired.Store(false)
	return func() {
		agent.apicClient = nil
		agent.cfg = nil
		agent.publishingLockAcquired.Store(false)
	}
}

func TestPublishingLockUnlock(t *testing.T) {
	defer setupDiscoveryTest(t)()

	assert.False(t, agent.publishingLockAcquired.Load())

	PublishingLock()
	assert.True(t, agent.publishingLockAcquired.Load())

	PublishingUnlock()
	assert.False(t, agent.publishingLockAcquired.Load())
}

// TestPublishAPI_InternalLocking verifies that PublishAPI acquires and releases
// the publishing lock internally when the caller has not pre-acquired it.
func TestPublishAPI_InternalLocking(t *testing.T) {
	defer setupDiscoveryTest(t)()

	publishCalled := false
	agent.apicClient = &mock.Client{
		PublishServiceMock: func(_ *apic.ServiceBody) (*management.APIService, error) {
			publishCalled = true
			return nil, nil
		},
	}

	assert.False(t, agent.publishingLockAcquired.Load())

	err := PublishAPI(apic.ServiceBody{})
	assert.Nil(t, err)
	assert.True(t, publishCalled)
	// flag is not set when PublishAPI manages the lock internally
	assert.False(t, agent.publishingLockAcquired.Load())
	// mutex must be released so it can be acquired again
	assert.True(t, agent.publishingLock.TryLock(), "lock should be released after PublishAPI returns")
	agent.publishingLock.Unlock()
}

// TestPublishAPI_WithPreAcquiredLock verifies that PublishAPI does not deadlock
// when called after PublishingLock() has already been acquired by the caller.
// Before this fix, PublishAPI would attempt to re-acquire the non-reentrant
// mutex and deadlock.
func TestPublishAPI_WithPreAcquiredLock(t *testing.T) {
	defer setupDiscoveryTest(t)()

	publishCalled := false
	agent.apicClient = &mock.Client{
		PublishServiceMock: func(_ *apic.ServiceBody) (*management.APIService, error) {
			publishCalled = true
			return nil, nil
		},
	}

	// External caller acquires the lock (as instancevalidator and eventsync do)
	PublishingLock()
	assert.True(t, agent.publishingLockAcquired.Load())

	// PublishAPI must not deadlock — it skips re-acquiring the already-held lock
	err := PublishAPI(apic.ServiceBody{})
	assert.Nil(t, err)
	assert.True(t, publishCalled)
	// flag remains true because the caller still holds the lock
	assert.True(t, agent.publishingLockAcquired.Load())

	PublishingUnlock()
	assert.False(t, agent.publishingLockAcquired.Load())
}
