package watchmanager

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
)

type proxyDialer interface {
	dial(ctx context.Context, addr string) (net.Conn, error)
}

type grpcProxyDialer struct {
	proxyAddress string
	userName     string
	password     string
}

func newGRPCProxyDialer(proxyURL *url.URL) proxyDialer {
	dialer := &grpcProxyDialer{
		proxyAddress: proxyURL.Host,
	}

	if user := proxyURL.User; user != nil {
		dialer.userName = user.Username()
		dialer.password, _ = user.Password()
	}
	return dialer
}

func (g *grpcProxyDialer) dial(ctx context.Context, addr string) (net.Conn, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", g.proxyAddress)
	if err != nil {
		return nil, err
	}

	err = g.proxyConnect(ctx, conn, addr)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func (g *grpcProxyDialer) proxyConnect(ctx context.Context, conn net.Conn, targetAddr string) error {
	req := g.createConnectRequest(ctx, targetAddr)
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

func (g *grpcProxyDialer) createConnectRequest(ctx context.Context, targetAddress string) *http.Request {
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Host: targetAddress},
	}

	if g.userName != "" {
		token := base64.StdEncoding.EncodeToString([]byte(g.userName + ":" + g.password))
		req.Header = map[string][]string{
			"Proxy-Authorization": {"Basic " + token},
		}
	}
	return req.WithContext(ctx)
}
