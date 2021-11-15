package stream

import (
	"fmt"
	"net/url"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"
)

// Client a client for opening up a grpc stream, and handling the received events on the stream.
type Client struct {
	errors           chan error
	events           chan *proto.Event
	handlers         []Handler
	host             string
	insecure         bool
	newClientManager wm.NewManagerFunc
	newEventManager  eventManagerFunc
	tenantID         string
	tlsConfig        config.TLSConfig
	tokenGetter      auth.TokenGetter
	topic            string
}

// NewClient creates a Client
func NewClient(
	host string,
	tenantID string,
	topic string,
	insecure bool,
	tlsConfig config.TLSConfig,
	tokenGetter auth.TokenGetter,
	handlers ...Handler,
) *Client {
	return &Client{
		errors:           make(chan error),
		events:           make(chan *proto.Event),
		handlers:         handlers,
		host:             host,
		insecure:         insecure,
		newClientManager: wm.New,
		newEventManager:  NewEventManager,
		tenantID:         tenantID,
		tlsConfig:        tlsConfig,
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
	ta := &auth.TokenAuth{
		TenantID:       sc.tenantID,
		TokenRequester: sc.tokenGetter,
	}

	u, _ := url.Parse(sc.host)

	cfg := &wm.Config{
		Host:        u.Host,
		Port:        443,
		TenantID:    sc.tenantID,
		TokenGetter: ta.GetToken,
	}

	manager, err := sc.newWatchManager(cfg)
	if err != nil {
		return err
	}

	client := api.NewClient(sc.tlsConfig, "")
	ric := NewResourceClient(fmt.Sprintf("%s/apis", sc.host), sc.tenantID, client, sc.tokenGetter)

	em := sc.newEventManager(
		sc.events,
		ric,
		sc.handlers...,
	)

	id, err := manager.RegisterWatch(sc.topic, sc.events, sc.errors)
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
