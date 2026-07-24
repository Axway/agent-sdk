package traceability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/event"
	"github.com/stretchr/testify/assert"
)

type fakeNetworkClient struct {
	connectErr error
	publishErr error
	closeCalls int
}

func (f *fakeNetworkClient) Connect() error { return f.connectErr }
func (f *fakeNetworkClient) Close() error {
	f.closeCalls++
	return nil
}
func (f *fakeNetworkClient) Publish(_ context.Context, _ event.Batch) error { return f.publishErr }
func (f *fakeNetworkClient) String() string                                { return "fake" }

func TestBackoffClientWaitOnError(t *testing.T) {
	init := 10 * time.Millisecond
	max := 100 * time.Millisecond

	tests := map[string]struct {
		publish        bool // false calls Connect, true calls Publish
		clientErr      error
		startDuration  time.Duration // 0 means the default (init*2)
		wantMinElapsed time.Duration
		wantMaxElapsed time.Duration
		wantDuration   time.Duration
		wantCloseCalls int
	}{
		"connect success does not sleep and resets duration": {
			wantMaxElapsed: init,
			wantDuration:   init * 2,
		},
		"connect error sleeps at least half the jittered duration and grows it": {
			clientErr:      errors.New("connect failed"),
			wantMinElapsed: init / 2,
			wantDuration:   init * 4,
		},
		"duration is capped at max": {
			clientErr:     errors.New("connect failed"),
			startDuration: max * 2, // simulate having already grown past max
			wantDuration:  max,
		},
		"publish error closes the inner client and grows duration": {
			publish:        true,
			clientErr:      errors.New("publish failed"),
			wantCloseCalls: 1,
			wantDuration:   init * 4,
		},
		"publish success does not close the inner client and resets duration": {
			publish:       true,
			startDuration: max,
			wantDuration:  init * 2,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			fake := &fakeNetworkClient{}
			if tc.publish {
				fake.publishErr = tc.clientErr
			} else {
				fake.connectErr = tc.clientErr
			}

			client := withBackoff(fake, init, max).(*backoffClient)
			if tc.startDuration > 0 {
				client.duration = tc.startDuration
			}

			start := time.Now()
			var err error
			if tc.publish {
				err = client.Publish(context.Background(), &MockBatch{})
			} else {
				err = client.Connect()
			}
			elapsed := time.Since(start)

			assert.Equal(t, tc.clientErr, err)
			assert.Equal(t, tc.wantDuration, client.duration)
			assert.Equal(t, tc.wantCloseCalls, fake.closeCalls)
			if tc.wantMinElapsed > 0 {
				assert.GreaterOrEqual(t, elapsed, tc.wantMinElapsed)
			}
			if tc.wantMaxElapsed > 0 {
				assert.Less(t, elapsed, tc.wantMaxElapsed)
			}
		})
	}
}

func TestBackoffClientClose(t *testing.T) {
	fake := &fakeNetworkClient{}
	client := withBackoff(fake, 10*time.Millisecond, 100*time.Millisecond).(*backoffClient)

	err := client.Close()
	assert.Nil(t, err)
	assert.Equal(t, 1, fake.closeCalls)
	_, open := <-client.done
	assert.False(t, open)
}

func TestBackoffClientString(t *testing.T) {
	fake := &fakeNetworkClient{}
	client := withBackoff(fake, 10*time.Millisecond, 100*time.Millisecond).(*backoffClient)
	assert.Equal(t, "backoff(fake)", client.String())
}
