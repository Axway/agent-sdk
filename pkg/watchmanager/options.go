package watchmanager

import (
	"crypto/tls"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// Option configures how we set up the watch connection.
type Option interface {
	apply(*watchOptions)
}

type keepAliveOption struct {
	time    time.Duration
	timeout time.Duration
}

type watchOptions struct {
	tlsCfg      *tls.Config
	keepAlive   keepAliveOption
	loggerEntry *logrus.Entry
}

type funcOption struct {
	f func(*watchOptions)
}

func (foptn *funcOption) apply(opt *watchOptions) {
	foptn.f(opt)
}

func newFuncOption(f func(*watchOptions)) Option {
	return &funcOption{
		f: f,
	}
}

// WithTLSConfig - sets up the TLS credentials or insecure if nil
func WithTLSConfig(tlsCfg *tls.Config) Option {
	return newFuncOption(func(o *watchOptions) {
		o.tlsCfg = tlsCfg
	})
}

// WithKeepAlive - configures keep alive ping interval and timeout to wait for ping ack
func WithKeepAlive(time, timeout time.Duration) Option {
	return newFuncOption(func(o *watchOptions) {
		o.keepAlive.time = time
		o.keepAlive.timeout = timeout
	})
}

// WithLogger - configures the logger to be used
func WithLogger(loggerEntry *logrus.Entry) Option {
	return newFuncOption(func(o *watchOptions) {
		o.loggerEntry = loggerEntry
	})
}

func (m *watchManager) appendRPCCredentialsOption(grpcDialOptions []grpc.DialOption) []grpc.DialOption {
	rpcCredential := newRPCAuth(m.tenantID, m.tokenGetter)
	return append(grpcDialOptions,
		grpc.WithPerRPCCredentials(rpcCredential),
	)
}

func (m *watchManager) appendTLSOption(grpcDialOptions []grpc.DialOption) []grpc.DialOption {
	if m.options.tlsCfg != nil {
		return append(grpcDialOptions, grpc.WithTransportCredentials(credentials.NewTLS(m.options.tlsCfg)))
	}
	return append(grpcDialOptions, grpc.WithInsecure())
}

func (m *watchManager) appendKeepAliveOption(grpcDialOptions []grpc.DialOption) []grpc.DialOption {
	return append(grpcDialOptions, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		PermitWithoutStream: false,
		Time:                m.options.keepAlive.time,
		Timeout:             m.options.keepAlive.timeout,
	}))
}

func (m *watchManager) appendLoggerOption(grpcDialOptions []grpc.DialOption) []grpc.DialOption {
	if m.options.loggerEntry != nil {
		return append(grpcDialOptions, grpc.WithStreamInterceptor(
			grpc_middleware.ChainStreamClient(
				func(entry *logrus.Entry) grpc.StreamClientInterceptor {
					opts := []grpc_logrus.Option{
						grpc_logrus.WithLevels(grpc_logrus.DefaultClientCodeToLevel),
						grpc_logrus.WithDurationField(grpc_logrus.DurationToDurationField),
					}
					return grpc_logrus.StreamClientInterceptor(entry, opts...)
				}(m.options.loggerEntry),
			),
		))
	}
	return grpcDialOptions
}
