package watchmanager

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/Axway/agent-sdk/pkg/harvester"
	"google.golang.org/grpc/connectivity"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// NewManagerFunc func signature to create a Manager
type NewManagerFunc func(cfg *Config, opts ...Option) (Manager, error)

// Manager - Interface to manage watch connections
type Manager interface {
	RegisterWatch(topic string, eventChan chan *proto.Event, errChan chan error) (string, error)
	CloseWatch(id string) error
	CloseConn()
	Status() bool
}

// TokenGetter - function to acquire token
type TokenGetter func() (string, error)

type watchManager struct {
	cfg                *Config
	clientMap          map[string]*watchClient
	connection         *grpc.ClientConn
	hClient            harvester.Harvest
	logger             log.FieldLogger
	mutex              sync.Mutex
	newWatchClientFunc newWatchClientFunc
	options            *watchOptions
}

// New - Creates a new watch manager
func New(cfg *Config, opts ...Option) (Manager, error) {
	err := cfg.validateCfg()
	if err != nil {
		return nil, err
	}

	logger := log.NewFieldLogger().
		WithComponent("watchManager").
		WithPackage("sdk.watchmanager")
	manager := &watchManager{
		cfg:                cfg,
		logger:             logger,
		clientMap:          make(map[string]*watchClient),
		options:            newWatchOptions(),
		newWatchClientFunc: proto.NewWatchClient,
	}

	for _, opt := range opts {
		opt.apply(manager.options)
	}

	manager.connection, err = manager.createConnection()
	if err != nil {
		manager.logger.
			WithError(err).
			Errorf("failed to establish connection with watch service")
	}

	seq := manager.options.sequenceProvider

	if seq != nil {
		harvesterConfig := &harvester.Config{
			Host:             manager.cfg.Host,
			Port:             manager.cfg.Port,
			TenantID:         manager.cfg.TenantID,
			TokenGetter:      manager.cfg.TokenGetter,
			ProxyURL:         manager.options.proxyURL,
			TLSCfg:           manager.options.tlsCfg,
			ClientTimeout:    manager.options.keepAlive.timeout,
			SequenceProvider: seq,
		}
		manager.hClient = harvester.NewClient(harvesterConfig)
	}

	return manager, err
}

func (m *watchManager) createConnection() (*grpc.ClientConn, error) {
	address := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	dialer, err := m.getDialer(address)
	if err != nil {
		return nil, err
	}

	grpcDialOptions := []grpc.DialOption{
		withKeepaliveParams(m.options.keepAlive.time, m.options.keepAlive.timeout),
		withRPCCredentials(m.cfg.TenantID, m.cfg.TokenGetter),
		withTLSConfig(m.options.tlsCfg),
		withDialer(dialer),
		chainStreamClientInterceptor(
			logrusStreamClientInterceptor(m.options.loggerEntry),
		),
	}

	m.logger.
		WithField("host", m.cfg.Host).
		WithField("port", m.cfg.Port).
		Infof("connecting to watch service")

	return grpc.Dial(address, grpcDialOptions...)
}

func (m *watchManager) getDialer(targetAddr string) (util.Dialer, error) {
	if m.options.singleEntryAddr == "" && m.options.proxyURL == "" {
		return nil, nil
	}
	var proxyURL *url.URL
	var err error
	if m.options.proxyURL != "" {
		proxyURL, err = url.Parse(m.options.proxyURL)
		if err != nil {
			return nil, err
		}
	}
	singleEntryHostMap := make(map[string]string)
	if m.options.singleEntryAddr != "" {
		singleEntryHostMap[targetAddr] = m.options.singleEntryAddr
	}
	return util.NewDialer(proxyURL, singleEntryHostMap), nil
}

// eventCatchUp - called until lastSequenceID is 0, caught up on events
func (m *watchManager) eventCatchUp(link string, events chan *proto.Event) error {
	if m.hClient == nil || m.options.sequenceProvider == nil {
		return nil
	}

	err := m.hClient.EventCatchUp(link, events)
	if err != nil {
		return err
	}

	return nil
}

// RegisterWatch - Registers a subscription with watch service using topic
func (m *watchManager) RegisterWatch(link string, events chan *proto.Event, errors chan error) (string, error) {
	client, err := newWatchClient(
		m.connection,
		clientConfig{
			errors:        errors,
			events:        events,
			tokenGetter:   m.cfg.TokenGetter,
			topicSelfLink: link,
		},
		m.newWatchClientFunc,
	)
	if err != nil {
		return "", err
	}

	subscriptionID, _ := uuid.NewUUID()
	subID := subscriptionID.String()

	m.mutex.Lock()
	m.clientMap[subID] = client
	m.mutex.Unlock()

	if err := m.eventCatchUp(link, events); err != nil {
		m.logger.WithError(err).Error("failed to sync events from harvester")
		client.cancelStreamCtx()
		if m.options.onEventSyncError != nil {
			m.options.onEventSyncError()
		}
		return subID, err
	}

	client.processRequest()
	go client.processEvents()

	m.logger.
		WithField("id", subID).
		WithField("watchtopic", link).
		Infof("registered watch client")

	return subID, nil
}

// CloseWatch closes the specified watch stream by id
func (m *watchManager) CloseWatch(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	client, ok := m.clientMap[id]
	if !ok {
		return errors.New("invalid watch subscription ID")
	}
	m.logger.WithField("watch-id", id).Info("closing connection for subscription")
	client.cancelStreamCtx()
	delete(m.clientMap, id)
	return nil
}

// CloseConn closes watch service connection, and all open streams
func (m *watchManager) CloseConn() {
	m.logger.Info("closing watch service connection")

	m.connection.Close()
	for id := range m.clientMap {
		delete(m.clientMap, id)
	}
}

// Status returns a boolean to indicate if the clients connected to central are active.
func (m *watchManager) Status() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ok := true

	if len(m.clientMap) == 0 {
		ok = false
	}

	for k, c := range m.clientMap {
		if !c.isRunning {
			m.logger.Debug("watch client is not running")
			ok = false
			delete(m.clientMap, k)
		}
	}

	return ok && m.connection.GetState() == connectivity.Ready
}
