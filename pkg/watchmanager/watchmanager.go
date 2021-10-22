package watchmanager

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// Manager - Interface to manage watch connection
type Manager interface {
	RegisterWatch(watchTopic string, eventChannel chan *proto.Event, errChannel chan error) (string, error)
	CloseWatch(watchSubscriptionID string) error
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

func defaultTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		},
	}
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
		logger:    logger,
		clientMap: make(map[string]*watchClient),
		options: &watchOptions{
			tlsCfg: defaultTLSConfig(),
			keepAlive: keepAliveOption{
				time:    30 * time.Second,
				timeout: 10 * time.Second,
			},
		},
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

	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = m.appendRPCCredentialsOption(grpcDialOptions)
	grpcDialOptions = m.appendTLSOption(grpcDialOptions)
	grpcDialOptions = m.appendKeepAliveOption(grpcDialOptions)
	grpcDialOptions = m.appendLoggerOption(grpcDialOptions)

	address := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	m.logger.WithField("host", m.cfg.Host).
		WithField("port", m.cfg.Port).
		Info("connecting to watch service")
	return grpc.Dial(address, grpcDialOptions...)
}

// RegisterWatch - Registers a subscription with watch service using topic
func (m *watchManager) RegisterWatch(watchTopicSelfLink string, eventChannel chan *proto.Event, errorChannel chan error) (string, error) {
	svcClient := proto.NewWatchServiceClient(m.connection)
	client, err := newWatchClient(svcClient, watchClientConfig{
		topicSelfLink: watchTopicSelfLink,
		tokenGetter:   m.cfg.TokenGetter,
		eventChannel:  eventChannel,
		errorChannel:  errorChannel})
	if err != nil {
		return "", err
	}

	subscriptionID, _ := uuid.NewUUID()
	m.clientMap[subscriptionID.String()] = client

	go client.processRequest()
	go client.processEvents()

	m.logger.WithField("watchtopic", watchTopicSelfLink).
		WithField("subscriptionId", subscriptionID.String()).
		Info("registered new watch client[subscription")
	return subscriptionID.String(), nil
}

func (m *watchManager) CloseWatch(subscriptionID string) error {
	m.logger.WithField("subscriptionId", subscriptionID).Info("closing watch")
	client, ok := m.clientMap[subscriptionID]
	if !ok {
		return errors.New("invalid watch subscription ID")
	}
	client.close()
	return nil
}

func (m *watchManager) Close() {
	m.logger.Info("closing watch service connection")
	// should trigger close on all open steams
	m.connection.Close()
}
