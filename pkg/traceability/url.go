package traceability

import (
	"fmt"
	"net"
	"net/url"
)

// buildURL replaces libbeat's common.MakeURL.
func buildURL(scheme, host string) (string, error) {
	if scheme == "" {
		scheme = "http"
	}
	if _, _, err := net.SplitHostPort(host); err != nil {
		host += ":443"
	}

	u, err := url.Parse(fmt.Sprintf("%s://%s/", scheme, host))
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
