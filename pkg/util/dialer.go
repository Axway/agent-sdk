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
)

// Dialer - interface for http dialer for proxy and single entry point
type Dialer interface {
	DialContext(ctx context.Context, network string, addr string) (net.Conn, error)
}

type dialer struct {
	singleEntryHostMap map[string]string
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

// DialContext - manages the connections to proxy and single entry point
func (d *dialer) DialContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	originalAddr := addr
	sniHost, ok := d.singleEntryHostMap[addr]
	if ok {
		addr = sniHost
	}
	if d.proxyAddress != "" {
		addr = d.proxyAddress
	}
	conn, err := (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 50 * time.Second,
		DualStack: true}).DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	if d.proxyAddress != "" {
		err = d.proxyConnect(ctx, conn, originalAddr, sniHost)
		if err != nil {
			conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

func (d *dialer) proxyConnect(ctx context.Context, conn net.Conn, targetAddr, sniHost string) error {
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
