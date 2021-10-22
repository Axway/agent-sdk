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
	cfg := &watchmanager.Config{
		Host:        config.Host,
		Port:        config.Port,
		TenantID:    config.TenantID,
		TokenGetter: ta.GetToken}
	wm, err := watchmanager.New(cfg, logger, watchOptions...)
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
	w.logger.Info("starting to watch events")

	eventChannel := make(chan *proto.Event)
	errCh := make(chan error)
	subscriptionID, err := w.wm.RegisterWatch(w.config.TopicSelfLink, eventChannel, errCh)
	if err != nil {
		w.logger.Error(err.Error())
		return
	}

	w.logger.
		WithField("subscriptionId", subscriptionID).Infof("watch registered for 30 minutes")
	wait := time.Duration(30 * time.Minute)
	for {
		select {
		case err = <-errCh:
			w.logger.
				WithField("subscriptionId", subscriptionID).
				Error(err.Error())
			return
		case <-time.After(wait):
			w.logger.
				WithField("subscriptionId", subscriptionID).Infof("initiating watch close")
			w.wm.CloseWatch(subscriptionID)
		case resourceEvent := <-eventChannel:
			w.logger.
				WithField("subscriptionId", subscriptionID).
				WithField("event", fmt.Sprintf("%+v", resourceEvent)).Infof("received message")
		}
	}
}
