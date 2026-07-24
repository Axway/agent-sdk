package traceability

import (
	"context"
	"math/rand"
	"time"

	"github.com/Axway/agent-sdk/pkg/event"
)

// backoffClient replaces libbeat's outputs.WithBackoff/backoff.EqualJitterBackoff.
type backoffClient struct {
	client NetworkClient
	init   time.Duration
	max    time.Duration
	done   chan struct{}

	duration time.Duration
}

func withBackoff(client NetworkClient, init, max time.Duration) NetworkClient {
	return &backoffClient{
		client: client,
		init:   init,
		max:    max,
		done:   make(chan struct{}),
		// allow sleeping at least the init period on the first wait
		duration: init * 2,
	}
}

func (b *backoffClient) Connect() error {
	err := b.client.Connect()
	b.waitOnError(err)
	return err
}

func (b *backoffClient) Close() error {
	err := b.client.Close()
	close(b.done)
	return err
}

func (b *backoffClient) Publish(ctx context.Context, batch event.Batch) error {
	err := b.client.Publish(ctx, batch)
	if err != nil {
		b.client.Close()
	}
	b.waitOnError(err)
	return err
}

func (b *backoffClient) String() string {
	return "backoff(" + b.client.String() + ")"
}

// waitOnError resets the backoff duration if err is nil, otherwise blocks for the
// jittered backoff duration and grows it for the next call.
func (b *backoffClient) waitOnError(err error) {
	if err == nil {
		b.duration = b.init * 2
		return
	}
	b.wait()
}

func (b *backoffClient) wait() {
	half := int64(b.duration / 2)
	sleep := time.Duration(half + rand.Int63n(half))

	b.duration *= 2
	if b.duration > b.max {
		b.duration = b.max
	}

	select {
	case <-b.done:
	case <-time.After(sleep):
	}
}
