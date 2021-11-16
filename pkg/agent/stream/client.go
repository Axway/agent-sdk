package stream

import (
	"fmt"
	"net/url"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/util/log"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"
)

// Client a client for opening up a grpc stream, and handling the received events on the stream.
type Client struct {
	apiClient        api.Client
	handlers         []Handler
	host             string
	insecure         bool
	newClientManager wm.NewManagerFunc
	newEventManager  eventManagerFunc
	tenantID         string
	tokenGetter      auth.TokenGetter
	topic            string
}

// NewClient creates a Client
func NewClient(
	host string,
	tenantID string,
	topic string,
	insecure bool,
	tokenGetter auth.TokenGetter,
	apiClient api.Client,
	handlers ...Handler,
) *Client {
	return &Client{
		apiClient:        apiClient,
		handlers:         handlers,
		host:             host,
		insecure:         insecure,
		newClientManager: wm.New,
		newEventManager:  NewEventManager,
		tenantID:         tenantID,
		tokenGetter:      tokenGetter,
		topic:            topic,
	}
}

func (sc *Client) newWatchManager(cfg *wm.Config) (wm.Manager, error) {
	logger := logrus.NewEntry(logrus.New())
	entry := logger.WithField("package", "client")

	var watchOptions []wm.Option
	watchOptions = append(watchOptions, wm.WithLogger(entry))
	if sc.insecure {
		watchOptions = append(watchOptions, wm.WithTLSConfig(nil))
	}

	return sc.newClientManager(cfg, logger, watchOptions...)
}

func (sc *Client) newStreamService() error {
	u, err := url.Parse(sc.host)
	if err != nil {
		return err
	}

	cfg := &wm.Config{
		Host:        u.Host,
		Port:        443,
		TenantID:    sc.tenantID,
		TokenGetter: sc.tokenGetter.GetToken,
	}

	manager, err := sc.newWatchManager(cfg)
	if err != nil {
		return err
	}

	ric := NewResourceClient(fmt.Sprintf("%s/apis", sc.host), sc.tenantID, sc.apiClient, sc.tokenGetter)

	events, errors := make(chan *proto.Event), make(chan error)

	em := sc.newEventManager(
		events,
		ric,
		sc.handlers...,
	)

	id, err := manager.RegisterWatch(sc.topic, events, errors)
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
