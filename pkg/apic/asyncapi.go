package apic

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	asyncSpec "github.com/swaggest/go-asyncapi/spec-2.4.0"
)

const (
	protocol = "protocol"
)

// custom parse URL to allow SASL_* schemes(url.Parse throws error)
func parseURL(strURL string) (scheme, host string, port int64, err error) {
	urlElements := strings.Split(strURL, "://")
	remaining := strURL
	if len(urlElements) > 1 {
		scheme = urlElements[0]
		remaining = urlElements[1]
	}

	strURL = fmt.Sprintf("tmp://%s", remaining)
	u, e := url.Parse(strURL)
	if e != nil {
		err = e
		return
	}

	host = u.Hostname()
	port, _ = strconv.ParseInt(u.Port(), 10, 32)
	return
}

type asyncApi struct {
	spec *asyncSpec.AsyncAPI
	raw  []byte
}

func (a *asyncApi) GetResourceType() string {
	return AsyncAPI
}

func (a *asyncApi) GetID() string {
	return a.spec.ID
}

func (a *asyncApi) GetTitle() string {
	return a.spec.Info.Title
}

func (a *asyncApi) GetVersion() string {
	return a.spec.Info.Version
}

func (a *asyncApi) GetEndpoints() ([]management.ApiServiceInstanceSpecEndpoint, error) {
	endpoints := make([]management.ApiServiceInstanceSpecEndpoint, 0)
	for _, server := range a.spec.Servers {
		scheme, host, port, err := parseURL(server.Server.URL)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, management.ApiServiceInstanceSpecEndpoint{
			Host:     host,
			Protocol: scheme,
			Port:     int32(port),
			Routing: management.ApiServiceInstanceSpecRouting{
				Details: map[string]interface{}{
					protocol: server.Server.Protocol,
				},
			},
		})
	}
	return endpoints, nil
}

func (a *asyncApi) GetSpecBytes() []byte {
	return a.raw
}
