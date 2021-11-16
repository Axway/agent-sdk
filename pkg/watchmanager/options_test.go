package watchmanager

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestWatchOptions(t *testing.T) {
	entry := logrus.NewEntry(logrus.New())
	opts := []Option{
		WithTLSConfig(nil),
		WithKeepAlive(1*time.Second, 1*time.Second),
		WithLogger(entry),
	}
	options := newWatchOptions()

	for _, opt := range opts {
		opt.apply(options)
	}

	assert.Nil(t, options.tlsCfg)
	assert.Equal(t, entry, options.loggerEntry)
	assert.Equal(t, 1*time.Second, options.keepAlive.timeout)
	assert.Equal(t, 1*time.Second, options.keepAlive.time)
}
