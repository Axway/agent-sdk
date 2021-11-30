package stream

import (
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// starter interface for starting a service
type starter interface {
	Start() error
}

// Client a client for creating a grpc stream, and handling the received events.
type Client struct {
	apiClient       api.Client
	apisHost        string
	handlers        []Handler
	manager         wm.Manager
	newEventManager eventManagerFunc
	tenantID        string
	tokenGetter     auth.TokenGetter
	topic           string
}

// NewClient creates a Client
func NewClient(
	host string,
	tenantID string,
	topic string,
	tokenGetter auth.TokenGetter,
	apiClient api.Client,
	manager wm.Manager,
	handlers ...Handler,
) *Client {
	return &Client{
		apiClient:       apiClient,
		handlers:        handlers,
		apisHost:        host,
		newEventManager: NewEventListener,
		tenantID:        tenantID,
		tokenGetter:     tokenGetter,
		topic:           topic,
		manager:         manager,
	}
}

func (sc *Client) newStreamService() error {
	ric := newResourceClient(sc.apisHost, sc.tenantID, sc.apiClient, sc.tokenGetter)

	events, errors := make(chan *proto.Event), make(chan error)

	em := sc.newEventManager(
		events,
		ric,
		sc.handlers...,
	)

	_, err := sc.manager.RegisterWatch(sc.topic, events, errors)
	if err != nil {
		return err
	}

	return em.Listen()
}

// Start starts the streaming client
func (sc *Client) Start() error {
	return sc.newStreamService()
}

// HealthCheck a health check endpoint for the connection to central.
func (sc *Client) HealthCheck() hc.CheckStatus {
	return func(_ string) *hc.Status {
		ok := sc.manager.Status()
		status := &hc.Status{
			Result: hc.OK,
		}

		if !ok {
			status.Result = hc.FAIL
			status.Details = "grpc client is not connected to central"
		}

		log.Tracef("grpc status: %s", status.Result)

		return status
	}
}

// Restart wraps a CheckStatus function and restarts the service if there is an error
func Restart(health hc.CheckStatus, starter starter) hc.CheckStatus {
	return func(name string) *hc.Status {
		status := health(name)

		if status.Result != hc.OK {
			go func() {
				log.Info("grpc-healthcheck: creating new grpc client")
				err := starter.Start()
				if err != nil {
					log.Errorf("grpc-healthcheck: failed to start the grpc client: %s", err)
				}
			}()
		}

		return status
	}
}
