package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

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
	seqCache cache.Cache
}

func (s *sequenceManager) GetSequence() int64 {
	cachedSeqID, err := s.seqCache.Get("watchSequenceID")
	if err == nil {
		if seqID, ok := cachedSeqID.(int64); ok {
			return seqID
		} else if seqID, ok := cachedSeqID.(float64); ok {
			return int64(seqID)
		}
	}
	return 0
}

func (s *sequenceManager) SetSequence(sequenceID int64) {
	s.seqCache.Set("watchSequenceID", sequenceID)
	s.seqCache.Save("sample.sequence")
}

// Todo - To be updated after cache persistence story
func getSequenceManager() *sequenceManager {
	seqCache := cache.New()
	err := seqCache.Load("sample.sequence")
	if err != nil {
		seqCache.Set("watchSequenceID", int64(1))
		seqCache.Save("sample.sequence")
	}

	return &sequenceManager{seqCache: seqCache}
}

func defaultTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		},
	}
}

// NewWatchClient creates a WatchClient
func NewWatchClient(config *Config, logger logrus.FieldLogger) (*WatchClient, error) {
	entry := logger.WithField("package", "client")

	var watchOptions []wm.Option
	watchOptions = []wm.Option{
		wm.WithLogger(entry),
	}

	if config.Insecure {
		watchOptions = append(watchOptions, wm.WithTLSConfig(nil))
	} else {
		watchOptions = append(watchOptions, wm.WithTLSConfig(defaultTLSConfig()))
	}

	ccfg := &corecfg.CentralConfiguration{
		URL:           fmt.Sprintf("https://%s:%d", config.Host, config.Port),
		ClientTimeout: 30 * time.Second,
		ProxyURL:      "",
		TenantID:      config.TenantID,
		TLS:           corecfg.NewTLSConfig(),
		GRPCCfg: corecfg.GRPCConfig{
			Enabled:  true,
			Insecure: config.Insecure,
			Host:     config.Host,
			Port:     int(config.Port),
		},
		Auth: &corecfg.AuthConfiguration{
			URL:        config.Auth.URL,
			PrivateKey: config.Auth.PrivateKey,
			PublicKey:  config.Auth.PublicKey,
			KeyPwd:     config.Auth.KeyPassword,
			Realm:      "Broker",
			ClientID:   config.Auth.ClientID,
			Timeout:    config.Auth.Timeout,
		},
	}
	ta := auth.NewPlatformTokenGetterWithCentralConfig(ccfg)
	hCfg := harvester.NewConfig(ccfg, ta, getSequenceManager())
	hClient := harvester.NewClient(hCfg)
	watchOptions = append(watchOptions, wm.WithHarvester(hClient, getSequenceManager()))
	cfg := &wm.Config{
		Host:        ccfg.GRPCCfg.Host,
		Port:        uint32(ccfg.GRPCCfg.Port),
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
