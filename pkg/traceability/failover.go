package traceability

import (
	"context"
	"math/rand"
	"strings"

	"github.com/Axway/agent-sdk/pkg/event"
)

// failoverClient replaces libbeat's outputs.failoverClient.
type failoverClient struct {
	clients []NetworkClient
	active  int
}

// newFailoverClient replaces libbeat's outputs.NewFailoverClient.
func newFailoverClient(clients []NetworkClient) NetworkClient {
	if len(clients) == 1 {
		return clients[0]
	}
	return &failoverClient{clients: clients, active: -1}
}

func (f *failoverClient) Connect() error {
	var next int
	active := f.active
	l := len(f.clients)

	switch {
	case l == 0:
		return ErrNoConnectionConfigured
	case l == 1:
		next = 0
	case l == 2 && active >= 0 && active <= 1:
		next = 1 - active
	default:
		for {
			// connect to a random server, to potentially spread the load when a
			// large number of agents with the same set of hosts start up at once
			next = rand.Int() % l
			if next != active {
				break
			}
		}
	}

	f.active = next
	return f.clients[next].Connect()
}

func (f *failoverClient) Close() error {
	if f.active < 0 {
		return ErrNoActiveConnection
	}
	return f.clients[f.active].Close()
}

func (f *failoverClient) Publish(ctx context.Context, batch event.Batch) error {
	if f.active < 0 {
		batch.Retry()
		return ErrNoActiveConnection
	}
	return f.clients[f.active].Publish(ctx, batch)
}

func (f *failoverClient) String() string {
	names := make([]string, len(f.clients))
	for i, client := range f.clients {
		names[i] = client.String()
	}
	return "failover(" + strings.Join(names, ",") + ")"
}
