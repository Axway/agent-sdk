package watchmanager

import (
	"crypto/tls"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

type keepAliveOption struct {
	time    time.Duration
	timeout time.Duration
}

// watchOptions options to use when creating a stream
type watchOptions struct {
	tlsCfg         *tls.Config
	keepAlive      keepAliveOption
	loggerEntry    *logrus.Entry
	sequenceGetter SequenceProvider
}

// newWatchOptions returns the default watchOptions
func newWatchOptions() *watchOptions {
	return &watchOptions{
		loggerEntry: logrus.NewEntry(logrus.New()),
		tlsCfg:      defaultTLSConfig(),
		keepAlive: keepAliveOption{
			time:    30 * time.Second,
			timeout: 10 * time.Second,
		},
	}
}

// WithTLSConfig - sets up the TLS credentials or insecure if nil
func WithTLSConfig(tlsCfg *tls.Config) Option {
	return funcOption(func(o *watchOptions) {
		o.tlsCfg = tlsCfg
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

// WithSyncEvents allows using the harvester client to sync events on watch registration
func WithSyncEvents(sequenceGetter SequenceProvider) Option {
	return funcOption(func(o *watchOptions) {
		o.sequenceGetter = sequenceGetter
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

	return grpc.WithInsecure()
}

// withKeepaliveParams sets the set keepalive parameters on the client-side
func withKeepaliveParams(time, timeout time.Duration) grpc.DialOption {
	return grpc.WithKeepaliveParams(
		keepalive.ClientParameters{
			PermitWithoutStream: false,
			Time:                time,
			Timeout:             timeout,
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
