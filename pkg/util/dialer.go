package util

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"golang.org/x/net/proxy"
	gr "google.golang.org/grpc/resolver"
)

const (
	// DefaultKeepAliveInterval - default duration to send keep alive pings
	DefaultKeepAliveInterval = 50 * time.Second
	// DefaultKeepAliveTimeout - default keepalive timeout
	DefaultKeepAliveTimeout = 10 * time.Second
)

// Dialer - interface for http dialer for proxy and single entry point
type Dialer interface {
	// Dial - interface used by libbeat for tcp network dial
	Dial(network string, addr string) (net.Conn, error)
	// DialContext - interface used by http transport
	DialContext(ctx context.Context, network string, addr string) (net.Conn, error)
	// GetProxyScheme() string
	GetProxyScheme() string
}

type dialer struct {
	singleEntryHostMap map[string]string
	proxyScheme        string
	proxyAddress       string
	userName           string
	password           string
}

// NewDialer - creates a new dialer
func NewDialer(proxyURL *url.URL, singleEntryHostMap map[string]string) Dialer {
	dialer := &dialer{
		singleEntryHostMap: singleEntryHostMap,
	}
	if proxyURL != nil {
		dialer.proxyScheme = proxyURL.Scheme
		dialer.proxyAddress = proxyURL.Host
		if user := proxyURL.User; user != nil {
			dialer.userName = user.Username()
			dialer.password, _ = user.Password()
		}
	}
	if dialer.singleEntryHostMap == nil {
		dialer.singleEntryHostMap = map[string]string{}
	}
	return dialer
}

// Dial- manages the connections to proxy and single entry point for tcp transports
func (d *dialer) Dial(network string, addr string) (net.Conn, error) {
	conn, err := d.DialContext(context.Background(), network, addr)
	if err == nil && len(d.singleEntryHostMap) > 0 && addr != conn.RemoteAddr().String() {
		log.Tracef("routing the traffic for %s via %s", addr, conn.RemoteAddr().String())
	}
	return conn, err
}

// DialContext - manages the connections to proxy and single entry point
func (d *dialer) DialContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	originalAddr := addr
	singleEntryHost, ok := d.singleEntryHostMap[addr]
	if ok {
		addr = singleEntryHost
	}
	if d.proxyAddress != "" {
		switch d.proxyScheme {
		case "socks5", "socks5h":
			return d.socksConnect(network, originalAddr, singleEntryHost)
		case "http", "https":
		default:
			return nil, fmt.Errorf("could not setup proxy, unsupported proxy scheme %s", d.proxyScheme)
		}
		addr = d.proxyAddress
	}
	conn, err := (&net.Dialer{
		Timeout:   DefaultKeepAliveTimeout,
		KeepAlive: DefaultKeepAliveInterval,
		DualStack: true}).DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	if d.proxyAddress != "" {
		switch d.proxyScheme {
		case "http", "https":
			err = d.httpConnect(ctx, conn, originalAddr, singleEntryHost)
			if err != nil {
				conn.Close()
				return nil, err
			}
		}
	}
	return conn, nil
}

func (d *dialer) GetProxyScheme() string {
	if d.proxyAddress != "" {
		return d.proxyScheme
	}
	return ""
}

func (d *dialer) socksConnect(network, addr, singleEntryHost string) (net.Conn, error) {
	var auth *proxy.Auth
	if d.userName != "" {
		auth = new(proxy.Auth)
		auth.User = d.userName
		if d.password != "" {
			auth.Password = d.password
		}
	}
	socksDialer, err := proxy.SOCKS5(network, d.proxyAddress, auth, nil)
	if err != nil {
		return nil, err
	}
	targetAddr := addr
	if singleEntryHost != "" {
		targetAddr = singleEntryHost
	}
	return socksDialer.Dial(network, targetAddr)
}

func (d *dialer) httpConnect(ctx context.Context, conn net.Conn, targetAddr, sniHost string) error {
	req := d.createConnectRequest(ctx, targetAddr, sniHost)
	if err := req.Write(conn); err != nil {
		return err
	}

	r := bufio.NewReader(conn)
	resp, err := http.ReadResponse(r, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to connect proxy, status : %s", resp.Status)
	}
	return nil
}

func (d *dialer) createConnectRequest(ctx context.Context, targetAddress, sniHost string) *http.Request {
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: targetAddress},
		Host:   targetAddress,
	}
	if sniHost != "" {
		req.URL = &url.URL{Opaque: sniHost}
	}

	if d.userName != "" {
		token := base64.StdEncoding.EncodeToString([]byte(d.userName + ":" + d.password))
		req.Header = map[string][]string{
			"Proxy-Authorization": {"Basic " + token},
		}
	}
	return req.WithContext(ctx)
}

type customGRPCResolverBuilder struct {
	addr      string
	authority string
	schema    string
}

func CreateCustomGRPCResolverBuilder(addr, authority, scheme string) gr.Builder {
	return &customGRPCResolverBuilder{
		addr:      addr,
		authority: authority,
		schema:    scheme,
	}
}

func (b *customGRPCResolverBuilder) Build(target gr.Target, cc gr.ClientConn, _ gr.BuildOptions) (gr.Resolver, error) {
	cc.UpdateState(gr.State{Endpoints: []gr.Endpoint{
		{
			Addresses: []gr.Address{
				{
					Addr:       b.addr,
					ServerName: b.authority,
				},
			},
		},
	}})
	return &nopResolver{}, nil
}

func (b *customGRPCResolverBuilder) Scheme() string {
	return b.schema
}

type nopResolver struct {
}

func (*nopResolver) ResolveNow(gr.ResolveNowOptions) {}

func (*nopResolver) Close() {}
