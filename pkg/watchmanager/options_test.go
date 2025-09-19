package watchmanager

import (
	"context"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type testSequenceProvider struct {
	id int64
}

func (s *testSequenceProvider) GetSequence() int64 {
	return s.id
}

func (s *testSequenceProvider) SetSequence(id int64) {
	s.id = id
}

func TestWatchOptions(t *testing.T) {
	entry := logrus.NewEntry(logrus.New())
	seq := &testSequenceProvider{}
	seq.SetSequence(1)
	opts := []Option{
		WithTLSConfig(nil),
		WithKeepAlive(1*time.Second, 1*time.Second),
		WithLogger(entry),
		WithHarvester(&mockHarvester{}, seq),
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

func (m mockHarvester) EventCatchUp(ctx context.Context, link string, events chan *proto.Event) error {
	// TODO implement me
	panic("implement me")
}

func (m mockHarvester) ReceiveSyncEvents(ctx context.Context, topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	// TODO implement me
	panic("implement me")
}
