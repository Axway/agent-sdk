package customunit

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	gr "google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 10240

type fakeQuotaEnforcementServer struct {
	customunits.UnimplementedQuotaEnforcementServer
}

type fakeCustomUnitMetricReportingServer struct {
	customunits.UnimplementedMetricReportingServiceServer
}

func Test_QuotaEnforcementInfo(t *testing.T) {

	ctx := context.Background()
	fakeServer := &fakeQuotaEnforcementServer{}
	client, _ := createQEConnection(fakeServer, ctx)
	response, err := client.QuotaEnforcementInfo()
	fmt.Println(err)
	assert.Nil(t, response)
}

func createQEConnection(fakeServer *fakeQuotaEnforcementServer, _ context.Context) (*customUnitClient, error) {
	lis := bufconn.Listen(bufSize)
	opt := grpc.WithContextDialer(
		func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		},
	)
	grpcServer := grpc.NewServer()
	customunits.RegisterQuotaEnforcementServer(grpcServer, fakeServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()
	quotaInfo := &customunits.QuotaInfo{
		ApiInfo: &customunits.APIInfo{
			ServiceName: "mockService",
		},
		AppInfo: &customunits.AppInfo{
			AppName: "mockApp",
		},
		Quota: &customunits.Quota{
			Unit: "mockUnit",
		},
	}
	cache := cache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	factory := NewCustomUnitClientFactory("bufnet", quotaInfo)
	return factory(cache, WithGRPCDialOption(opt))

}

func Test_MetricReporting(t *testing.T) {
	ctx := context.Background()
	fakeServer := &fakeCustomUnitMetricReportingServer{}
	client, _ := createMRConnection(fakeServer, ctx)
	metricReportChan := make(chan *customunits.MetricReport, 100)
	go client.StartMetricReporting(metricReportChan)

	time.Sleep(5 * time.Second)
	client.Stop()
}

func createMRConnection(fakeServer *fakeCustomUnitMetricReportingServer, _ context.Context) (*customUnitClient, error) {
	lis := bufconn.Listen(bufSize)
	opt := grpc.WithContextDialer(
		func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		},
	)
	grpcServer := grpc.NewServer()
	customunits.RegisterMetricReportingServiceServer(grpcServer, fakeServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()

	cache := cache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	factory := NewCustomUnitClientFactory("bufnet", &customunits.QuotaInfo{})
	return factory(cache, WithGRPCDialOption(opt), WithGRPCDialOption(grpc.WithResolvers(&builder{url: "bufnet"})))
}

type builder struct {
	url string
}

func (b *builder) Build(target gr.Target, cc gr.ClientConn, _ gr.BuildOptions) (gr.Resolver, error) {
	cc.UpdateState(gr.State{Endpoints: []gr.Endpoint{
		{
			Addresses: []gr.Address{
				{
					Addr: b.url,
				},
			},
		},
	}})
	return &nopResolver{}, nil
}
func (b *builder) Scheme() string {
	return ""
}

type nopResolver struct {
}

func (*nopResolver) ResolveNow(gr.ResolveNowOptions) {}
func (*nopResolver) Close()                          {}
