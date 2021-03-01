package transaction

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/config"
	metrics "github.com/rcrowley/go-metrics"
)

func createCentralCfg(url, env string) *config.CentralConfiguration {

	cfg := config.NewCentralConfig(config.DiscoveryAgent).(*config.CentralConfiguration)
	cfg.URL = url
	cfg.TenantID = "123456"
	cfg.Environment = env
	authCfg := cfg.Auth.(*config.AuthConfiguration)
	authCfg.URL = url + "/auth"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "DOSA_1111"
	authCfg.PrivateKey = "../transaction/testdata/private_key.pem"
	authCfg.PublicKey = "../transaction/testdata/public_key"
	return cfg
}

var accessToken = `eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0IiwiaWF0IjoxNjE0NjA0NzE0LCJleHAiOjE2NDYxNDA3MTQsImF1ZCI6InRlc3RhdWQiLCJzdWIiOiIxMjM0NTYiLCJvcmdfZ3VpZCI6IjEyMzQtMTIzNC0xMjM0LTEyMzQifQ.5Uqt0oFhMgmI-sLQKPGkHwknqzlTxv-qs9I_LmZ18LQ`

func TestMetricCollector(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"" + accessToken + "\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/gatekeeper") {
			body, _ := ioutil.ReadAll(req.Body)
			fmt.Println("Gatekeeper Req : " + string(body))
		}

	}))

	defer s.Close()
	cfg := createCentralCfg(s.URL, "demo")
	// cfg.GatekeeperURL = "https://engvncn8usbzk.x.pipedream.net/"
	cfg.GatekeeperURL = s.URL + "/gatekeeper"
	agent.Initialize(cfg)
	eventChannel := make(chan interface{}, 1028)
	metricCollector = &collector{
		lock:               &sync.Mutex{},
		registry:           metrics.NewRegistry(),
		apiMetricMap:       make(map[string]*APIMetric),
		apiStatusMetricMap: make(map[string]map[string]*StatusMetric),
		eventChannel:       eventChannel,
	}
	CreatePublisher(eventChannel)
	startTime := time.Now()
	metricCollector.collectMetric("111", "111", "200", 10, "", "")
	metricCollector.collectMetric("111", "111", "200", 20, "", "")
	metricCollector.collectMetric("111", "111", "200", 30, "", "")
	metricCollector.collectMetric("111", "111", "401", 10, "", "")
	metricCollector.collectMetric("111", "111", "401", 20, "", "")

	metricCollector.collectMetric("222", "222", "200", 5, "", "")
	metricCollector.collectMetric("222", "222", "200", 5, "", "")

	metricCollector.generateAggregation(startTime, time.Now())
	time.Sleep(30 * time.Second)

	startTime = time.Now()
	metricCollector.apiMetricMap = make(map[string]*APIMetric)
	metricCollector.apiStatusMetricMap = make(map[string]map[string]*StatusMetric)

	metricCollector.collectMetric("111", "111", "200", 5, "", "")
	metricCollector.collectMetric("111", "111", "200", 15, "", "")
	metricCollector.collectMetric("111", "111", "401", 15, "", "")
	metricCollector.collectMetric("111", "111", "401", 5, "", "")
	metricCollector.collectMetric("111", "111", "401", 120, "", "")

	metricCollector.collectMetric("222", "222", "200", 5, "", "")
	metricCollector.collectMetric("222", "222", "200", 50, "", "")
	metricCollector.collectMetric("222", "222", "400", 15, "", "")

	metricCollector.generateAggregation(time.Now(), time.Now())
	time.Sleep(30 * time.Second)
}
