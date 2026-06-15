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

func TestPublishAPI_Locking(t *testing.T) {
	tests := []struct {
		name           string
		preAcquireLock bool
	}{
		{
			// PublishAPI acquires and releases the lock internally when the caller
			// has not pre-acquired it.
			name:           "internal locking",
			preAcquireLock: false,
		},
		{
			// PublishAPI must not deadlock when the caller has already acquired the
			// lock. Before this fix, PublishAPI would attempt to re-acquire the
			// non-reentrant mutex and deadlock.
			name:           "pre-acquired lock",
			preAcquireLock: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer setupDiscoveryTest(t)()

			publishCalled := false
			agent.apicClient = &mock.Client{
				PublishServiceMock: func(_ *apic.ServiceBody) (*management.APIService, error) {
					publishCalled = true
					return nil, nil
				},
			}

			if tc.preAcquireLock {
				PublishingLock()
				assert.True(t, agent.publishingLockAcquired.Load())
			} else {
				assert.False(t, agent.publishingLockAcquired.Load())
			}

			err := PublishAPI(apic.ServiceBody{})
			assert.Nil(t, err)
			assert.True(t, publishCalled)

			if tc.preAcquireLock {
				// flag remains true because the caller still holds the lock
				assert.True(t, agent.publishingLockAcquired.Load())
				PublishingUnlock()
				assert.False(t, agent.publishingLockAcquired.Load())
			} else {
				// flag is not set when PublishAPI manages the lock internally
				assert.False(t, agent.publishingLockAcquired.Load())
				// mutex must be released so it can be acquired again
				assert.True(t, agent.publishingLock.TryLock(), "lock should be released after PublishAPI returns")
				agent.publishingLock.Unlock()
			}
		})
	}
}
