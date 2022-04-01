package watchmanager

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	"google.golang.org/grpc/connectivity"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"

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

// SequenceProvider - Interface to provide event sequence ID to harvester client to fetch events
type SequenceProvider interface {
	GetSequence() int64
}

// TokenGetter - function to acquire token
type TokenGetter func() (string, error)

type watchManager struct {
	cfg                *Config
	clientMap          map[string]*watchClient
	connection         *grpc.ClientConn
	hClient            *harvesterClient
	logger             logrus.FieldLogger
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

	entry := logrus.NewEntry(log.Get())

	manager := &watchManager{
		cfg:                cfg,
		logger:             entry.WithField("package", "watchmanager"),
		clientMap:          make(map[string]*watchClient),
		options:            newWatchOptions(),
		newWatchClientFunc: proto.NewWatchClient,
	}

	for _, opt := range opts {
		opt.apply(manager.options)
	}

	manager.connection, err = manager.createConnection()
	if err != nil {
		log.Errorf("failed to establish connection with watch service: %s", err.Error())
	}

	if manager.options.sequenceGetter != nil {
		harvesterConfig := &harvesterConfig{
			host:        manager.cfg.Host,
			port:        manager.cfg.Port,
			tenantID:    manager.cfg.TenantID,
			tokenGetter: manager.cfg.TokenGetter,
			proxyURL:    manager.options.proxyURL,
			tlsCfg:      manager.options.tlsCfg,
		}
		manager.hClient = newHarvesterClient(harvesterConfig)
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

	log.Infof("connecting to watch service. host: %s. port: %d", m.cfg.Host, m.cfg.Port)

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

// RegisterWatch - Registers a subscription with watch service using topic
func (m *watchManager) RegisterWatch(link string, events chan *proto.Event, errors chan error) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

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

	m.clientMap[subID] = client

	client.processRequest()

	var lastSequenceID int64
	var sequenceID int64
	if m.hClient != nil && m.options.sequenceGetter != nil {
		sequenceID = m.options.sequenceGetter.GetSequence()
		if sequenceID > 0 {
			var err error
			lastSequenceID, err = m.hClient.receiveSyncEvents(link, sequenceID, events)
			if err != nil {
				client.handleError(err)
				return subID, err
			}
		}
	}

	if lastSequenceID > 0 {
		// wait for all current sequences to be processed before processing new ones
		for sequenceID < lastSequenceID {
			sequenceID = m.options.sequenceGetter.GetSequence()
		}
	}

	go client.processEvents()

	log.Infof("registered watch client. id: %s. watchtopic: %s", subID, link)

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
	log.Infof("closing watch for subscription: %s", id)
	client.cancelStreamCtx()
	delete(m.clientMap, id)
	return nil
}

// CloseConn closes watch service connection, and all open streams
func (m *watchManager) CloseConn() {
	log.Info("closing watch service connection")

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
			log.Debugf("watchmanager: watch client is not running.")
			ok = false
			delete(m.clientMap, k)
		}
	}

	return ok && m.connection.GetState() == connectivity.Ready
}
