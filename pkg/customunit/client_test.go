package customunit

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 10240

type fakeQuotaEnforcementServer struct {
	customunits.UnimplementedQuotaEnforcementServer
}

type fakeCustomUnitMetricReportingServer struct {
	customunits.UnimplementedQuotaEnforcementServer
}

func Test_QuotaEnforcementInfo(t *testing.T) {

	ctx := context.Background()
	fakeServer := &fakeQuotaEnforcementServer{}
	client, _ := createQEConnection(fakeServer, ctx)
	response, err := client.QuotaEnforcementInfo()
	fmt.Println(err)
	assert.Nil(t, response)
}

func createQEConnection(fakeServer *fakeQuotaEnforcementServer, ctx context.Context) (customUnitClient, error) {
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
	factory := NewCustomUnitClientFactory("bufnet", cache, quotaInfo)
	streamCtx, streamCancel := context.WithCancel(context.Background())
	return factory(streamCtx, streamCancel, WithGRPCDialOption(opt))

}

func Test_MetricReporting(t *testing.T) {

	ctx := context.Background()
	fakeServer := &fakeCustomUnitMetricReportingServer{}
	client, _ := createMRConnection(fakeServer, ctx)
	err := client.MetricReporting()
	fmt.Println(err)
}

func createMRConnection(fakeServer *fakeCustomUnitMetricReportingServer, ctx context.Context) (customUnitClient, error) {
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

	cache := cache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	factory := NewCustomUnitClientFactory("bufnet", cache, &customunits.QuotaInfo{})
	streamCtx, streamCancel := context.WithCancel(context.Background())
	return factory(streamCtx, streamCancel, WithGRPCDialOption(opt))

}
