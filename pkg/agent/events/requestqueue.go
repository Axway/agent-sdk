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
	cancel    context.CancelFunc
	ctx       context.Context
	logger    log.FieldLogger
	requestCh chan *proto.Request
	receiveCh chan *proto.Request
	isActive  bool
	lock      *sync.Mutex
}

// NewRequestQueueFunc type for creating a new request queue
type NewRequestQueueFunc func(requestCh chan *proto.Request) RequestQueue

// NewRequestQueue creates a new queue for the requests to be sent for watch subscription
func NewRequestQueue(requestCh chan *proto.Request) RequestQueue {
	ctx, cancel := context.WithCancel(context.Background())
	logger := log.NewFieldLogger().
		WithComponent("requestQueue").
		WithPackage("sdk.agent.events")

	return &requestQueue{
		cancel:    cancel,
		ctx:       ctx,
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
	return q.isActive
}

func (q *requestQueue) Write(request *proto.Request) error {
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
	q.lock.Lock()
	defer q.lock.Unlock()

	go func() {
		q.isActive = true
		defer func() {
			q.isActive = false
		}()

		for {
			if q.process() {
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
