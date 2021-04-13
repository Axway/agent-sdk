package chimera

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

type ProxyResolverFunc func(*http.Request) (*url.URL, error)

// Options for creating a Chimera client.
type ClientOptions struct {
	Protocol      Scheme
	Host          string
	AuthKey       string
	TLSConfig     *tls.Config
	ProxyResolver ProxyResolverFunc
	Timeout       time.Duration
}

// Options for publishing message.
type PublishOptions struct {
	Headers map[string]string
}

// Options for subscribing to Chimera.
type SubscribeOptions struct {
	Headers     map[string]string
	Resubscribe int
}

func (opts ClientOptions) getTimeout() time.Duration {
	timeout := 5 * time.Second // default
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}
	return timeout
}

func (opts ClientOptions) getTransport() *http.Transport {
	if opts.TLSConfig != nil || opts.ProxyResolver != nil {
		transport := &http.Transport{
			TLSClientConfig: opts.TLSConfig,
			Proxy:           opts.ProxyResolver,
		}
		return transport
	}
	return nil
}
