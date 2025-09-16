package events

import (
	"context"
	"errors"
	"sync"

	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type RequestQueue interface {
	Start()
	Write(request *proto.Request) error
	Stop()
	IsActive() bool
}

// requestQueue
type requestQueue struct {
	ctx       context.Context
	cancel    context.CancelFunc
	logger    log.FieldLogger
	requestCh chan *proto.Request
	receiveCh chan *proto.Request
	isActive  bool
	lock      *sync.Mutex
}

// NewRequestQueueFunc type for creating a new request queue
type NewRequestQueueFunc func(ctx context.Context, cancel context.CancelFunc, requestCh chan *proto.Request) RequestQueue

// NewRequestQueue creates a new queue for the requests to be sent for watch subscription
func NewRequestQueue(ctx context.Context, cancel context.CancelFunc, requestCh chan *proto.Request) RequestQueue {
	logger := log.NewFieldLogger().
		WithComponent("requestQueue").
		WithPackage("sdk.agent.events")

	return &requestQueue{
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
		requestCh: requestCh,
		receiveCh: make(chan *proto.Request, 1),
		lock:      &sync.Mutex{},
	}
}

func (q *requestQueue) Stop() {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.cancel != nil {
		q.cancel()
	}
}

func (q *requestQueue) IsActive() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.isActive
}

func (q *requestQueue) Write(request *proto.Request) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	if !q.isActive {
		return errors.New("request queue is not active")
	}

	if q.receiveCh != nil {
		q.logger.WithField("requestType", request.RequestType).Trace("received stream request")
		q.receiveCh <- request
	}
	return nil
}

func (q *requestQueue) Start() {
	go func() {
		q.lock.Lock()
		q.isActive = true
		q.lock.Unlock()

		defer func() {
			q.lock.Lock()
			defer q.lock.Unlock()
			q.isActive = false
		}()

		for {
			if q.process() {
				break
			}
			if q.ctx.Err() != nil {
				break
			}
		}
	}()
}

func (q *requestQueue) process() bool {
	done := false
	select {
	case req := <-q.receiveCh:
		q.logger.WithField("requestType", req.RequestType).Trace("forwarding stream request")
		q.requestCh <- req
		q.logger.WithField("requestType", req.RequestType).Trace("stream request forwarded")
	case <-q.ctx.Done():
		q.logger.Trace("stream request queue has been gracefully stopped")
		done = true
		if q.receiveCh != nil {
			close(q.receiveCh)
			q.receiveCh = nil
		}
		break
	}

	return done
}
