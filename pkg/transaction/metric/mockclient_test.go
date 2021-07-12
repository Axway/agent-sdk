package metric

import (
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/elastic/beats/v7/libbeat/outputs"
	beatPub "github.com/elastic/beats/v7/libbeat/publisher"
)

type MockClient struct {
	outputs.NetworkClient

	retry    int
	pubCount int
}

func (m *MockClient) Close() error   { return nil }
func (m *MockClient) Connect() error { return nil }
func (m *MockClient) Publish(batch beatPub.Batch) error {
	m.pubCount++
	switch {
	case m.retry >= m.pubCount:
		batch.Retry()
	default:
		batch.ACK()
	}
	return nil
}
func (m *MockClient) String() string {
	return ""
}

var myMockClient outputs.Client

func mockGetClient() (*traceability.Client, error) {
	tpClient := &traceability.Client{}
	tpClient.SetTransportClient(myMockClient)
	return tpClient, nil
}

func setupMockClient(retries int) {
	myMockClient = &MockClient{
		pubCount: 0,
		retry:    retries,
	}
	traceability.GetClient = mockGetClient
}
