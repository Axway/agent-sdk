package events

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type RequestQueue interface {
	Write(request *proto.Request) error
	Start() chan error
	Stop()
	IsActive() bool
}

// requestQueue
type requestQueue struct {
	cancel    context.CancelFunc
	ctx       context.Context
	handlers  []handler.Handler
	logger    log.FieldLogger
	requestCh chan *proto.Request
	receiveCh chan *proto.Request
	isActive  bool
}

// NewRequestQueueFunc type for creating a new request queue
type NewRequestQueueFunc func(
	requestCh chan *proto.Request, cbs ...handler.Handler,
) RequestQueue

// NewRequestQueue creates a new queue for the requests to be sent for watch subscription
func NewRequestQueue(
	requestCh chan *proto.Request, cbs ...handler.Handler,
) RequestQueue {
	ctx, cancel := context.WithCancel(context.Background())
	logger := log.NewFieldLogger().
		WithComponent("RequestQueue").
		WithPackage("sdk.agent.events")

	return &requestQueue{
		cancel:    cancel,
		ctx:       ctx,
		handlers:  cbs,
		logger:    logger,
		requestCh: requestCh,
		isActive:  true,
		receiveCh: make(chan *proto.Request, 1),
	}
}

// Stop stops the request queue
func (em *requestQueue) Stop() {
	if em != nil {
		em.isActive = false
		em.cancel()
	}
}

func (em *requestQueue) Write(request *proto.Request) error {
	if em.receiveCh != nil {
		em.receiveCh <- request
	}
	return nil
}

func (em *requestQueue) IsActive() bool {
	if em != nil {
		return em.isActive
	}
	return false
}
func (em *requestQueue) Start() chan error {
	errCh := make(chan error)
	go func() {
		for {
			done, err := em.start()
			if done && err == nil {
				errCh <- nil
				break
			}

			if err != nil {
				errCh <- err
				break
			}
		}
	}()

	return errCh
}

func (em *requestQueue) start() (done bool, err error) {
	select {
	case req := <-em.receiveCh:
		em.requestCh <- req
	case <-em.ctx.Done():
		em.logger.Trace("stream request queue has been gracefully stopped")
		done = true
		err = nil
		if em.receiveCh != nil {
			close(em.receiveCh)
			em.receiveCh = nil
		}
		break
	}

	return done, err
}
