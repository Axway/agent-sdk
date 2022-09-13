package client

import (
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/harvester"

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

type sequenceManager struct {
	seqCache       cache.Cache
	watchTopicName string
}

func (s *sequenceManager) GetSequence() int64 {
	cachedSeqID, err := s.seqCache.Get("watchSequenceID")
	if err == nil {
		if seqID, ok := cachedSeqID.(float64); ok {
			return int64(seqID)
		}
	}
	return 0
}

func (s *sequenceManager) SetSequence(sequenceID int64) {
	s.seqCache.Set("watchSequenceID", sequenceID)
}

// Todo - To be updated after cache persistence story
func getSequenceManager() *sequenceManager {
	seqCache := cache.New()
	err := seqCache.Load("sample.sequence")
	if err != nil {
		seqCache.Set("watchSequenceID", int64(0))
		seqCache.Save("sample.sequence")
	}

	return &sequenceManager{seqCache: seqCache}
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

	ccfg := &corecfg.CentralConfiguration{
		URL:           config.Host,
		ClientTimeout: 15,
		ProxyURL:      "",
		TenantID:      config.TenantID,
		TLS:           corecfg.NewTLSConfig(),
		GRPCCfg: corecfg.GRPCConfig{
			Enabled:  true,
			Insecure: config.Insecure,
		},
	}

	hCfg := harvester.NewConfig(ccfg, ta, getSequenceManager())
	hClient := harvester.NewClient(hCfg)
	watchOptions = append(watchOptions, wm.WithHarvester(hClient, getSequenceManager()))

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

	log = log.WithField("subscription-id", subscriptionID)
	log.Infof("watch registered successfully")

	for {
		select {
		case err = <-errCh:
			log.Error(err)
			w.wm.CloseWatch(subscriptionID)
			return
		case event := <-eventChannel:
			bts, _ := json.MarshalIndent(event, "", "  ")
			log.Info(string(bts))
		}
	}
}
