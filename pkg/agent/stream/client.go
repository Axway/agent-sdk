package stream

import (
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/util/log"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// Client a client for opening up a grpc stream, and handling the received events on the stream.
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
		newEventManager: NewEventManager,
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

	id, err := sc.manager.RegisterWatch(sc.topic, events, errors)
	if err != nil {
		return err
	}

	log.Debugf("watch-controller subscription-id: %s", id)

	return em.Start()
}

// Start starts the streaming client
func (sc *Client) Start() error {
	return sc.newStreamService()
}
