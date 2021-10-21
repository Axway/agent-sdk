package watchmanager

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// Manager - Interface to manage watch connection
type Manager interface {
	RegisterWatch(watchTopic string, eventChannel chan *proto.Event, errChannel chan error) (string, error)
}

// TokenGetter - function to acquire token
type TokenGetter func() (string, error)

type watchManager struct {
	host        string
	port        uint32
	tenantID    string
	tokenGetter TokenGetter
	clientMap   map[string]*watchClient
	connection  *grpc.ClientConn
	options     *watchOptions
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
func New(host string, port uint32, tenantID string, tokenGetter TokenGetter, opts ...Option) (Manager, error) {
	manager := &watchManager{
		host:        host,
		port:        port,
		clientMap:   make(map[string]*watchClient),
		tenantID:    tenantID,
		tokenGetter: tokenGetter,
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

	var err error
	manager.connection, err = manager.createConnection()
	return manager, err
}

func (m *watchManager) createConnection() (*grpc.ClientConn, error) {
	address := fmt.Sprintf("%s:%d", m.host, m.port)

	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = m.appendRPCCredentialsOption(grpcDialOptions)
	grpcDialOptions = m.appendTLSOption(grpcDialOptions)
	grpcDialOptions = m.appendKeepAliveOption(grpcDialOptions)
	grpcDialOptions = m.appendLoggerOption(grpcDialOptions)

	return grpc.Dial(address, grpcDialOptions...)
}

// RegisterWatch - Registers a subscription with watch service using topic
func (m *watchManager) RegisterWatch(watchTopicSelfLink string, eventChannel chan *proto.Event, errorChannel chan error) (string, error) {
	svcClient := proto.NewWatchServiceClient(m.connection)
	stream, err := svcClient.CreateWatch(context.Background())
	if err != nil {
		return "", err
	}

	client := newWatchClient(watchTopicSelfLink, m.tokenGetter, stream)
	uuiduuid, _ := uuid.NewUUID()
	m.clientMap[uuiduuid.String()] = client

	go client.processRequest(errorChannel)
	go client.processEvents(eventChannel, errorChannel)

	return uuiduuid.String(), nil
}
