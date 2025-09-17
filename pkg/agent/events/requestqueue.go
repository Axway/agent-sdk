package events

import (
	"context"
	"errors"
	"sync/atomic"

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
	isActive  atomic.Bool
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
		isActive:  atomic.Bool{},
	}
}

func (q *requestQueue) Stop() {
	if !q.isActive.Load() {
		return
	}

	defer q.isActive.Store(false)
	if q.cancel != nil {
		q.cancel()
		close(q.receiveCh)
	}
}

func (q *requestQueue) IsActive() bool {
	return q.isActive.Load()
}

func (q *requestQueue) Write(request *proto.Request) error {
	if !q.isActive.Load() {
		return errors.New("request queue is not active")
	}

	q.logger.WithField("requestType", request.RequestType).Trace("received stream request")
	q.receiveCh <- request
	return nil
}

func (q *requestQueue) Start() {
	go func() {
		log.Info("------- starting request queue")
		defer log.Info("------- request queue stopped")
		q.isActive.Store(true)
		defer q.isActive.Store(false)

		for {
			if q.process() {
				q.Stop()
				break
			}
		}
	}()
}

func (q *requestQueue) process() bool {
	select {
	case req := <-q.receiveCh:
		if q.ctx.Err() != nil {
			return true
		}
		q.logger.WithField("requestType", req.RequestType).Info("------- forwarding stream request")
		q.requestCh <- req
		q.logger.WithField("requestType", req.RequestType).Info("------- stream request forwarded")
		return false
	case <-q.ctx.Done():
		q.logger.Info("------- stream request queue has been gracefully stopped")
		return true
	}
}
