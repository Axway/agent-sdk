package metric

import (
	"context"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/outputs"
	beatPub "github.com/elastic/beats/v7/libbeat/publisher"
)

type MockClient struct {
	outputs.NetworkClient

	retry       int
	pubCount    int
	eventsAcked int
	published   bool
}

func (m *MockClient) Close() error   { return nil }
func (m *MockClient) Connect() error { m.published = false; return nil }
func (m *MockClient) Publish(_ context.Context, batch beatPub.Batch) error {
	m.published = false
	m.pubCount++
	switch {
	case m.retry >= m.pubCount:
		batch.Retry()
	case m.retry < m.pubCount && m.retry > 3:
		return fmt.Errorf("")
	default:
		m.eventsAcked = len(batch.Events())
		batch.ACK()
	}
	m.published = true
	return nil
}
func (m *MockClient) String() string {
	return ""
}

var myMockClient outputs.Client

func mockGetClient() (*traceability.Client, error) {
	tpClient := &traceability.Client{}
	tpClient.SetTransportClient(myMockClient)
	tpClient.SetLogger(log.NewFieldLogger())
	return tpClient, nil
}

func setupMockClient(retries int) {
	myMockClient = &MockClient{
		pubCount:    0,
		retry:       retries,
		eventsAcked: 0,
	}
	traceability.GetClient = mockGetClient
}
