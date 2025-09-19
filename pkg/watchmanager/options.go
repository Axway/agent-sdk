package watchmanager

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/sirupsen/logrus"
)

// Option configures how we set up the watch connection.
type Option interface {
	apply(*watchOptions)
}

// funcOption defines a func that receives a watchOptions. Implements the Option interface.
type funcOption func(*watchOptions)

// apply calls the original func to update the watchOptions.
func (f funcOption) apply(opt *watchOptions) {
	f(opt)
}

type eventSyncCb func()

type keepAliveOption struct {
	time    time.Duration
	timeout time.Duration
}

// watchOptions options to use when creating a stream
type watchOptions struct {
	ctx              context.Context
	cancel           context.CancelFunc
	tlsCfg           *tls.Config
	proxyURL         string
	singleEntryAddr  string
	keepAlive        keepAliveOption
	loggerEntry      *logrus.Entry
	sequence         events.SequenceProvider
	onEventSyncError eventSyncCb
	harvester        harvester.Harvest
	requestCh        chan *proto.Request
}

// newWatchOptions returns the default watchOptions
func newWatchOptions() *watchOptions {
	// Default context and cancel function
	ctx, cancel := context.WithCancel(context.Background())

	return &watchOptions{
		ctx:         ctx,
		cancel:      cancel,
		loggerEntry: logrus.NewEntry(log.Get()),
		tlsCfg:      defaultTLSConfig(),
		keepAlive: keepAliveOption{
			time:    util.DefaultKeepAliveInterval,
			timeout: util.DefaultKeepAliveTimeout,
		},
	}
}

// WithEventSyncError - callback func to invoke when there is an error syncing initial events
func WithEventSyncError(f eventSyncCb) Option {
	return funcOption(func(o *watchOptions) {
		o.onEventSyncError = f
	})
}

// WithTLSConfig - sets up the TLS credentials or insecure if nil
func WithTLSConfig(tlsCfg *tls.Config) Option {
	return funcOption(func(o *watchOptions) {
		o.tlsCfg = tlsCfg
	})
}

// WithProxy - sets up the proxy to be used
func WithProxy(proxy string) Option {
	return funcOption(func(o *watchOptions) {
		o.proxyURL = proxy
	})
}

// WithSingleEntryAddr - sets up the single entry host to be used
func WithSingleEntryAddr(singleEntryAddr string) Option {
	return funcOption(func(o *watchOptions) {
		o.singleEntryAddr = singleEntryAddr
	})
}

// WithKeepAlive - sets keep alive ping interval and timeout to wait for ping ack
func WithKeepAlive(time, timeout time.Duration) Option {
	return funcOption(func(o *watchOptions) {
		o.keepAlive.time = time
		o.keepAlive.timeout = timeout
	})
}

// WithLogger sets the logger to be used by the client, overriding the default logger
func WithLogger(loggerEntry *logrus.Entry) Option {
	return funcOption(func(o *watchOptions) {
		o.loggerEntry = loggerEntry
	})
}

// WithHarvester allows using the harvester client to sync events on watch registration
func WithHarvester(harvester harvester.Harvest, sequence events.SequenceProvider) Option {
	return funcOption(func(o *watchOptions) {
		o.sequence = sequence
		o.harvester = harvester
	})
}

// WithRequestChannel - sets up the channel to send requests for watch subscriptions
func WithRequestChannel(requestCh chan *proto.Request) Option {
	return funcOption(func(o *watchOptions) {
		o.requestCh = requestCh
	})
}

func WithContext(ctx context.Context, cancel context.CancelFunc) Option {
	return funcOption(func(o *watchOptions) {
		o.ctx = ctx
		o.cancel = cancel
	})
}

// withRPCCredentials sets credentials and places auth state on each outbound RPC.
func withRPCCredentials(tenantID string, tokenGetter TokenGetter) grpc.DialOption {
	rpcCredential := newRPCAuth(tenantID, tokenGetter)
	return grpc.WithPerRPCCredentials(rpcCredential)
}

// withTLSConfig configures connection level security credentials
func withTLSConfig(tlsCfg *tls.Config) grpc.DialOption {
	if tlsCfg != nil {
		return grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
	}
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}

// withKeepaliveParams sets the set keepalive parameters on the client-side
func withKeepaliveParams(time, timeout time.Duration) grpc.DialOption {
	return grpc.WithKeepaliveParams(
		keepalive.ClientParameters{
			PermitWithoutStream: true,
			Time:                time,
			Timeout:             timeout,
		})
}

// withDialer sets up the proxy dialer
func withDialer(dialer util.Dialer) grpc.DialOption {
	if dialer == nil {
		return &grpc.EmptyDialOption{}
	}

	return grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", addr)
	})
}

// logrusStreamClientInterceptor returns a new streaming client interceptor that optionally logs the execution of gRPC calls
func logrusStreamClientInterceptor(entry *logrus.Entry) grpc.StreamClientInterceptor {
	opts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(grpc_logrus.DefaultClientCodeToLevel),
		grpc_logrus.WithDurationField(grpc_logrus.DurationToDurationField),
	}

	return grpc_logrus.StreamClientInterceptor(entry, opts...)
}

// chainStreamClientInterceptor returns a DialOption that specifies the interceptor for streaming RPCs
func chainStreamClientInterceptor(interceptors ...grpc.StreamClientInterceptor) grpc.DialOption {
	return grpc.WithStreamInterceptor(
		grpc_middleware.ChainStreamClient(interceptors...),
	)
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
