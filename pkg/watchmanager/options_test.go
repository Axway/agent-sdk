package watchmanager

import (
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type testSequenceProvider struct{}

func (s *testSequenceProvider) GetSequence() int64 {
	return 0
}

func (s *testSequenceProvider) SetSequence(_ int64) {
}

func TestWatchOptions(t *testing.T) {
	entry := logrus.NewEntry(logrus.New())
	opts := []Option{
		WithTLSConfig(nil),
		WithKeepAlive(1*time.Second, 1*time.Second),
		WithLogger(entry),
		WithHarvester(&mockHarvester{}, &testSequenceProvider{}),
		WithProxy("http://proxy"),
		WithSingleEntryAddr("single-entry"),
	}

	options := newWatchOptions()

	for _, opt := range opts {
		opt.apply(options)
	}

	assert.Nil(t, options.tlsCfg)
	assert.Equal(t, entry, options.loggerEntry)
	assert.Equal(t, 1*time.Second, options.keepAlive.timeout)
	assert.Equal(t, 1*time.Second, options.keepAlive.time)
	assert.NotNil(t, options.sequence)
	assert.Equal(t, "http://proxy", options.proxyURL)
	assert.Equal(t, "single-entry", options.singleEntryAddr)
}

type mockHarvester struct{}

func (m mockHarvester) EventCatchUp(link string, events chan *proto.Event) error {
	// TODO implement me
	panic("implement me")
}

func (m mockHarvester) ReceiveSyncEvents(topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	// TODO implement me
	panic("implement me")
}
