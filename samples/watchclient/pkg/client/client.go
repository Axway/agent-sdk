package client

import (
	"encoding/json"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic/auth"

	"github.com/sirupsen/logrus"

	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// WatchClient - stream client for connecting to the Watch Controller
type WatchClient struct {
	config *Config
	logger logrus.FieldLogger
	wm     wm.Manager
}

// NewWatchClient creates a WatchClient
func NewWatchClient(config *Config, logger logrus.FieldLogger) (*WatchClient, error) {
	entry := logger.WithField("package", "client")

	var watchOptions []wm.Option
	watchOptions = append(watchOptions, wm.WithLogger(entry))
	if config.Insecure {
		watchOptions = append(watchOptions, wm.WithTLSConfig(nil))
	}

	ta := auth.NewTokenAuth(config.Auth, config.TenantID)

	cfg := &wm.Config{
		Host:        config.Host,
		Port:        config.Port,
		TenantID:    config.TenantID,
		TokenGetter: ta.GetToken,
	}

	w, err := wm.New(cfg, watchOptions...)
	if err != nil {
		return nil, err
	}

	return &WatchClient{
		config: config,
		logger: entry,
		wm:     w,
	}, nil
}

// Watch starts a two-way stream with the Watch Controller
func (w WatchClient) Watch() {
	log := w.logger
	log.Info("starting to watch events")

	eventChannel, errCh := make(chan *proto.Event), make(chan error)
	subscriptionID, err := w.wm.RegisterWatch(w.config.TopicSelfLink, eventChannel, errCh)
	if err != nil {
		log.Error(err)
		return
	}

	log = log.WithField("subscriptionId", subscriptionID)
	log.Infof("watch registered successfully")

	wait := 30 * time.Minute
	for {
		select {
		case err = <-errCh:
			log.Error(err)
			return
		case <-time.After(wait):
			log.Infof("initiating watch close")

			w.wm.CloseWatch(subscriptionID)
		case event := <-eventChannel:
			bts, _ := json.MarshalIndent(event, "", "  ")

			log.Info(string(bts))
		}
	}
}
