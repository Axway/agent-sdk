package watchmanager

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/connectivity"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// Manager - Interface to manage watch connections
type Manager interface {
	RegisterWatch(topic string, eventChan chan *proto.Event, errChan chan error) (string, error)
	CloseWatch(id string) error
	Close()
	Status() bool
}

// TokenGetter - function to acquire token
type TokenGetter func() (string, error)

type watchManager struct {
	cfg                *Config
	clientMap          map[string]*watchClient
	connection         *grpc.ClientConn
	options            *watchOptions
	logger             logrus.FieldLogger
	newWatchClientFunc newWatchClientFunc
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
	return manager, err
}

func (m *watchManager) createConnection() (*grpc.ClientConn, error) {
	grpcDialOptions := []grpc.DialOption{
		withKeepaliveParams(m.options.keepAlive.time, m.options.keepAlive.timeout),
		withRPCCredentials(m.cfg.TenantID, m.cfg.TokenGetter),
		withTLSConfig(m.options.tlsCfg),
		chainStreamClientInterceptor(
			logrusStreamClientInterceptor(m.options.loggerEntry),
		),
	}

	address := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	log.Infof("connecting to watch service. host: %s. port: %d", m.cfg.Host, m.cfg.Port)

	return grpc.Dial(address, grpcDialOptions...)
}

// RegisterWatch - Registers a subscription with watch service using topic
func (m *watchManager) RegisterWatch(link string, events chan *proto.Event, errors chan error) (string, error) {
	client, err := newWatchClient(
		m.connection,
		clientConfig{
			topicSelfLink: link,
			tokenGetter:   m.cfg.TokenGetter,
			events:        events,
			errors:        errors,
		},
		m.newWatchClientFunc,
	)
	if err != nil {
		return "", err
	}

	subscriptionID, _ := uuid.NewUUID()
	subID := subscriptionID.String()

	m.clientMap[subID] = client

	go client.processRequest()
	go client.processEvents()

	log.Infof("registered watch client. id: %s. watchtopic: %s", subID, link)

	return subID, nil
}

// CloseWatch closes the specified watch stream by id
func (m *watchManager) CloseWatch(id string) error {
	log.Infof("closing watch for subscription: %s", id)
	client, ok := m.clientMap[id]
	if !ok {
		return errors.New("invalid watch subscription ID")
	}
	client.cancelStream()
	delete(m.clientMap, id)
	return nil
}

// Close - Close the watch service connection, and all open streams
func (m *watchManager) Close() {
	log.Info("closing watch service connection")

	m.connection.Close()
	for id := range m.clientMap {
		delete(m.clientMap, id)
	}
}

// Status returns a boolean to indicate if the clients connected to central are active.
func (m *watchManager) Status() bool {
	for _, c := range m.clientMap {
		if c.isRunning == false {
			log.Debugf("watch client is not running")
			return false
		}
	}

	return m.connection.GetState() == connectivity.Ready
}
