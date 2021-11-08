package watchmanager

import (
	"errors"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// Manager - Interface to manage watch connection
type Manager interface {
	RegisterWatch(topic string, eventChan chan *proto.Event, errChan chan error) (string, error)
	CloseWatch(id string) error
	Close()
}

// TokenGetter - function to acquire token
type TokenGetter func() (string, error)

type watchManager struct {
	cfg        *Config
	clientMap  map[string]*watchClient
	connection *grpc.ClientConn
	options    *watchOptions
	logger     logrus.FieldLogger
}

// New - Creates a new watch manager
func New(cfg *Config, logger logrus.FieldLogger, opts ...Option) (Manager, error) {
	err := cfg.validateCfg()
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = logrus.New()
	}

	manager := &watchManager{
		cfg:       cfg,
		logger:    logger.WithField("package", "watchmanager"),
		clientMap: make(map[string]*watchClient),
		options:   newWatchOptions(),
	}

	for _, opt := range opts {
		opt.apply(manager.options)
	}

	manager.connection, err = manager.createConnection()
	if err != nil {
		logger.Errorf("failed to establish connection with watch service: %s", err.Error())
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
	m.logger.WithField("host", m.cfg.Host).
		WithField("port", m.cfg.Port).
		Info("connecting to watch service")

	return grpc.Dial(address, grpcDialOptions...)
}

// RegisterWatch - Registers a subscription with watch service using topic
func (m *watchManager) RegisterWatch(link string, events chan *proto.Event, errors chan error) (string, error) {
	client, err := newWatchClient(
		m.connection,
		watchClientConfig{
			topicSelfLink: link,
			tokenGetter:   m.cfg.TokenGetter,
			eventChannel:  events,
			errorChannel:  errors,
		},
	)
	if err != nil {
		return "", err
	}

	subscriptionID, _ := uuid.NewUUID()
	subID := subscriptionID.String()

	m.clientMap[subID] = client

	go client.processRequest()
	go client.processEvents()

	m.logger.WithField("watchtopic", link).
		WithField("subscriptionId", subID).
		Info("registered new watch client[subscription]")

	return subID, nil
}

// CloseWatch closes the specified watch stream by id
func (m *watchManager) CloseWatch(id string) error {
	m.logger.WithField("subscriptionId", id).Info("closing watch")
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
	m.logger.Info("closing watch service connection")

	m.connection.Close()
	for id := range m.clientMap {
		delete(m.clientMap, id)
	}
}
