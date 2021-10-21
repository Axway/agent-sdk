package client

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// WatchClient - stream client for connecting to the Watch Controller
type WatchClient struct {
	config *Config
	logger logrus.FieldLogger
	wm     watchmanager.Manager
}

// NewWatchClient creates a WatchClient
func NewWatchClient(config *Config, logger logrus.FieldLogger) (*WatchClient, error) {
	entry := logger.WithField("package", "client")

	var watchOptions []watchmanager.Option
	if config.Insecure {
		watchOptions = append(watchOptions, watchmanager.WithTLSConfig(nil))
	}

	watchOptions = append(watchOptions,
		watchmanager.WithKeepAlive(30*time.Second, 10*time.Second))

	watchOptions = append(watchOptions,
		watchmanager.WithLogger(entry))

	ta := newTokenAuth(config.Auth, config.TenantID)
	wm, err := watchmanager.New(config.Host, config.Port, config.TenantID, ta.GetToken, watchOptions...)
	if err != nil {
		return nil, err
	}

	return &WatchClient{
		config: config,
		logger: entry,
		wm:     wm,
	}, nil
}

// Watch starts a two-way stream with the Watch Controller
func (w WatchClient) Watch() {
	w.logger.Info("opening the CreateWatch stream")

	eventChannel := make(chan *proto.Event)
	errCh := make(chan error)
	subscriptionID, err := w.wm.RegisterWatch(w.config.TopicSelfLink, eventChannel, errCh)
	if err != nil {
		w.logger.Error(err.Error())
		return
	}

	w.logger.
		WithField("subscription-id", subscriptionID).Infof("Watch registered")

	for {
		select {
		case err = <-errCh:
			w.logger.
				WithField("subscription-id", subscriptionID).
				Error(err.Error())
			return
		case resourceEvent := <-eventChannel:
			w.logger.
				WithField("subscription-id", subscriptionID).
				WithField("event", fmt.Sprintf("%+v", resourceEvent)).Infof("received message")
		}
	}
}
