package traceability

import (
	"context"
	"testing"

	"github.com/Axway/agent-sdk/pkg/event"
	"github.com/stretchr/testify/assert"
)

func TestNewFailoverClientSingleClient(t *testing.T) {
	fake := &fakeNetworkClient{}
	client := newFailoverClient([]NetworkClient{fake})
	assert.Same(t, NetworkClient(fake), client)
}

func TestFailoverClientBeforeAnyConnect(t *testing.T) {
	tests := map[string]struct {
		clients []NetworkClient
		op      func(client NetworkClient) error
		wantErr error
	}{
		"connect with no clients configured": {
			clients: nil,
			op:      func(client NetworkClient) error { return client.Connect() },
			wantErr: ErrNoConnectionConfigured,
		},
		"publish before any connect": {
			clients: []NetworkClient{&fakeNetworkClient{}, &fakeNetworkClient{}},
			op:      func(client NetworkClient) error { return client.Publish(context.Background(), &MockBatch{}) },
			wantErr: ErrNoActiveConnection,
		},
		"close before any connect": {
			clients: []NetworkClient{&fakeNetworkClient{}, &fakeNetworkClient{}},
			op:      func(client NetworkClient) error { return client.Close() },
			wantErr: ErrNoActiveConnection,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client := newFailoverClient(tc.clients)
			assert.Equal(t, tc.wantErr, tc.op(client))
		})
	}
}

func TestFailoverClientNeverRepeatsActive(t *testing.T) {
	tests := map[string]struct {
		numClients int
	}{
		"two clients":   {numClients: 2},
		"three clients": {numClients: 3},
		"four clients":  {numClients: 4},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			clients := make([]NetworkClient, tc.numClients)
			for i := range clients {
				clients[i] = &fakeNetworkClient{}
			}
			client := newFailoverClient(clients).(*failoverClient)

			// the first connect picks randomly since there's no active client yet;
			// every connect after that must never repeat the previously active client
			for i := 0; i < 50; i++ {
				prevActive := client.active
				assert.Nil(t, client.Connect())
				if prevActive >= 0 {
					assert.NotEqual(t, prevActive, client.active)
				}
			}
		})
	}
}

func TestFailoverClientPublishAndCloseRouteToActiveOnly(t *testing.T) {
	c0 := &fakeNetworkClient{}
	c1 := &fakeNetworkClient{}
	client := newFailoverClient([]NetworkClient{c0, c1}).(*failoverClient)

	client.active = 0
	assert.Nil(t, client.Publish(context.Background(), &MockBatch{}))
	assert.Nil(t, client.Close())
	assert.Equal(t, 1, c0.closeCalls)
	assert.Equal(t, 0, c1.closeCalls)
}

func TestFailoverClientString(t *testing.T) {
	client := newFailoverClient([]NetworkClient{&fakeNetworkClient{}, &fakeNetworkClient{}}).(*failoverClient)
	assert.Equal(t, "failover(fake,fake)", client.String())
}

var _ event.Batch = (*MockBatch)(nil)
